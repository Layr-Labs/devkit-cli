package container

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	templcmd "github.com/Layr-Labs/devkit-cli/pkg/commands/template"
	"github.com/Layr-Labs/devkit-cli/pkg/common"
	"github.com/urfave/cli/v2"
)

// DeployCommand: devkit container deploy [app] [--dev, -d]
var DeployCommand = &cli.Command{
	Name:  "deploy",
	Usage: "Build, push, deploy to TEE (app optional in project dir)",
	Flags: append([]cli.Flag{
		// Deployment target
		&cli.BoolFlag{Name: "dev", Aliases: []string{"d"}, Usage: "Use dev TEE environment"},
		// Image
		&cli.StringFlag{Name: "tag", Usage: "Image tag", Value: "latest"},
		&cli.StringFlag{Name: "registry", Usage: "Container registry host", Value: "ghcr.io"},
		&cli.StringFlag{Name: "owner", Usage: "Registry owner/org (e.g. gh username or org)"},
		&cli.StringFlag{Name: "repo", Usage: "Full repo path override (e.g. ghcr.io/org/app)"},
		// On-chain
		&cli.StringFlag{Name: "controller", Usage: "AppController address (0x...)"},
		&cli.StringFlag{Name: "app-address", Usage: "Existing application address (0x...) for publishRelease"},
		&cli.StringFlag{Name: "rpc-url", Usage: "Ethereum RPC URL (overrides env)"},
		&cli.UintFlag{Name: "upgrade-by-time", Usage: "Unix timestamp by which operators must upgrade"},
		// Env payload for ReleaseWithEnv
		&cli.StringFlag{Name: "env-hex", Usage: "Hex-encoded encrypted env (0x prefixed)"},
		&cli.StringFlag{Name: "env-file", Usage: "Path to encrypted env file (hex or binary)"},
		// Tooling
		&cli.StringFlag{Name: "cast-bin", Usage: "Path to foundry cast binary", Value: "cast"},
	}, common.GlobalFlags...),
	Action: func(cCtx *cli.Context) error {
		logger := common.LoggerFromContext(cCtx)

		// 1) Resolve app name (image name)
		appName := cCtx.Args().Get(0)
		if appName == "" {
			n, _, _, _, _ := templcmd.GetTemplateInfo()
			appName = n
		}
		if appName == "" {
			return fmt.Errorf("unable to determine app name; pass [app] or run in a project directory")
		}

		// 2) Delegate to project script if present and executable
		script := filepath.Join(".devkit", "scripts", "container", "deploy")
		if fi, err := os.Stat(script); err == nil && fi.Mode()&0111 != 0 {
			logger.Debug("Found project deploy script, delegating to it")
			_, err := common.CallTemplateScript(
				cCtx.Context,
				logger,
				"",
				script,
				common.ExpectNonJSONResponse,
				[]byte("--image"), []byte(appName),
				[]byte("--tag"), []byte(cCtx.String("tag")),
				[]byte("--dev"), []byte(fmt.Sprintf("%v", cCtx.Bool("dev"))),
			)
			if err != nil {
				return fmt.Errorf("deploy (script) failed: %w", err)
			}
			logger.Info("Deployed container %s via project script", appName)
			return nil
		}

		// 3) Built-in fallback flow: docker build/login/push + cast publishRelease
		logger.Debug("No project deploy script found; using built-in fallback")

		// Registry/auth
		registry := cCtx.String("registry")
		owner := cCtx.String("owner")
		if owner == "" {
			owner = os.Getenv("GHCR_OWNER")
		}
		if owner == "" {
			return fmt.Errorf("required: --owner or GHCR_OWNER env for registry auth")
		}
		tag := cCtx.String("tag")
		repo := cCtx.String("repo")
		if repo == "" {
			repo = fmt.Sprintf("%s/%s/%s", registry, owner, appName)
		}

		// Build
		if err := runCmd(cCtx, "docker", "build", "-t", fmt.Sprintf("%s:%s", repo, tag), "."); err != nil {
			return fmt.Errorf("docker build failed: %w", err)
		}

		// Login (stdin token)
		token := os.Getenv("GHCR_TOKEN")
		if token == "" {
			return fmt.Errorf("required: GHCR_TOKEN env for registry auth")
		}
		if err := runWithStdin(cCtx, token+"\n", "docker", "login", registry, "-u", owner, "--password-stdin"); err != nil {
			return fmt.Errorf("docker login failed: %w", err)
		}

		// Push
		if err := runCmd(cCtx, "docker", "push", fmt.Sprintf("%s:%s", repo, tag)); err != nil {
			return fmt.Errorf("docker push failed: %w", err)
		}

		// Resolve digest
		out, err := runCapture(cCtx, "docker", "inspect", "--format", "{{index .RepoDigests 0}}", fmt.Sprintf("%s:%s", repo, tag))
		if err != nil {
			return fmt.Errorf("resolve digest failed: %w", err)
		}
		imageURI := strings.TrimSpace(out)
		if imageURI == "" || !strings.Contains(imageURI, "@") {
			return fmt.Errorf("unexpected image digest: %q", imageURI)
		}

		// On-chain inputs
		controller := valOrEnv(cCtx.String("controller"), "APP_CONTROLLER")
		if controller == "" {
			return fmt.Errorf("required: --controller or APP_CONTROLLER env")
		}
		appAddr := valOrEnv(cCtx.String("app-address"), "APP_ADDRESS")
		if appAddr == "" {
			return fmt.Errorf("required: --app-address or APP_ADDRESS env (createApp not implemented in fallback)")
		}
		rpcURL := firstNonEmpty(cCtx.String("rpc-url"), getenvPreferred(cCtx.Bool("dev"), "ETH_RPC_URL_DEV", "ETH_RPC_URL"))
		if rpcURL == "" {
			return fmt.Errorf("required: --rpc-url or ETH_RPC_URL(_DEV) env")
		}
		privKey := getenvPreferred(cCtx.Bool("dev"), "PUBLISHER_KEY_DEV", "PUBLISHER_KEY")
		if privKey == "" {
			return fmt.Errorf("required: PUBLISHER_KEY(_DEV) env for transaction signing")
		}

		envHex := strings.TrimSpace(cCtx.String("env-hex"))
		if envHex == "" && cCtx.String("env-file") != "" {
			b, rdErr := os.ReadFile(cCtx.String("env-file"))
			if rdErr != nil {
				return fmt.Errorf("read env-file failed: %w", rdErr)
			}
			s := strings.TrimSpace(string(b))
			if strings.HasPrefix(s, "0x") {
				envHex = s
			} else {
				enc := make([]byte, hex.EncodedLen(len(b)))
				hex.Encode(enc, b)
				envHex = "0x" + string(enc)
			}
		}
		if envHex == "" {
			envHex = "0x"
		}

		// Derive Artifact(digest, registry) from image digest and repo
		parts := strings.Split(imageURI, "@")
		if len(parts) != 2 {
			return fmt.Errorf("unexpected image uri format: %s", imageURI)
		}
		algoAndHex := parts[1]
		if !strings.HasPrefix(algoAndHex, "sha256:") {
			return fmt.Errorf("unsupported digest algorithm in %s", algoAndHex)
		}
		hexPart := strings.TrimPrefix(algoAndHex, "sha256:")
		if len(hexPart) != 64 {
			return fmt.Errorf("invalid digest length: %d", len(hexPart))
		}
		digestBytes32 := "0x" + hexPart

		// upgradeByTime is required for Release
		upgradeBy := cCtx.Uint("upgrade-by-time")
		if upgradeBy == 0 {
			return fmt.Errorf("required: --upgrade-by-time (unix timestamp)")
		}

		castBin := cCtx.String("cast-bin")

		// publishRelease(IApp app, ReleaseWithEnv release)
		// ReleaseWithEnv: ( (Artifact[] artifacts, uint32 upgradeByTime), bytes env )
		// Artifact: (bytes32 digest, string registry)
		// Cast ABI: publishRelease(address,((bytes32,string)[],uint32),bytes)
		artifactsArg := fmt.Sprintf("[(%s,\"%s\")]", digestBytes32, repo)
		releaseTuple := fmt.Sprintf("(%s,%d)", artifactsArg, upgradeBy)

		if err := runCmd(
			cCtx,
			castBin,
			"send",
			controller,
			"publishRelease(address,((bytes32,string)[],uint32),bytes)",
			appAddr,
			releaseTuple,
			envHex,
			"--rpc-url", rpcURL,
			"--private-key", privKey,
		); err != nil {
			return fmt.Errorf("publishRelease failed: %w", err)
		}

		logger.Info("Deployed %s:%s -> %s", appName, tag, imageURI)
		return nil
	},
}

func runCmd(cCtx *cli.Context, name string, args ...string) error {
	cmd := exec.CommandContext(cCtx.Context, name, args...)
	cmd.Stdout, cmd.Stderr = os.Stdout, os.Stderr
	return cmd.Run()
}

func runWithStdin(cCtx *cli.Context, stdin string, name string, args ...string) error {
	cmd := exec.CommandContext(cCtx.Context, name, args...)
	cmd.Stdout, cmd.Stderr = os.Stdout, os.Stderr
	cmd.Stdin = bytes.NewBufferString(stdin)
	return cmd.Run()
}

func runCapture(cCtx *cli.Context, name string, args ...string) (string, error) {
	cmd := exec.CommandContext(cCtx.Context, name, args...)
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return "", err
	}
	return out.String(), nil
}

func valOrEnv(v, env string) string {
	if v != "" {
		return v
	}
	return os.Getenv(env)
}

func getenvPreferred(useDev bool, devKey, prodKey string) string {
	if useDev {
		if v := os.Getenv(devKey); v != "" {
			return v
		}
	}
	return os.Getenv(prodKey)
}

func firstNonEmpty(v ...string) string {
	for _, s := range v {
		if s != "" {
			return s
		}
	}
	return ""
}


