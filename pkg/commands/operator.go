package commands

import (
	"fmt"

	"github.com/urfave/cli/v2"

	"github.com/Layr-Labs/devkit-cli/pkg/operator"
)

const operatorDataPath = "config/contexts/operators.json"

// OperatorCommands returns the operator command group.
func OperatorCommands() *cli.Command {
	return &cli.Command{
		Name:  "operator",
		Usage: "Manage operator strategies and magnitudes",
		Subcommands: []*cli.Command{
			{
				Name:   "list",
				Usage:  "List all operators and their strategies",
				Action: operatorListCmd,
			},
			{
				Name:  "strategy",
				Usage: "Manage operator strategies",
				Subcommands: []*cli.Command{
					{
						Name:  "list",
						Usage: "List all strategies for an operator",
						Flags: []cli.Flag{
							&cli.StringFlag{Name: "operator-id", Required: true},
						},
						Action: strategyListCmd,
					},
					{
						Name:  "get",
						Usage: "Show allocations for a strategy",
						Flags: []cli.Flag{
							&cli.StringFlag{Name: "operator-id", Required: true},
							&cli.StringFlag{Name: "strategy", Required: true},
						},
						Action: strategyGetCmd,
					},
					{
						Name:  "allocate",
						Usage: "Allocate magnitude to an operator set",
						Flags: []cli.Flag{
							&cli.StringFlag{Name: "operator-id", Required: true},
							&cli.StringFlag{Name: "strategy", Required: true},
							&cli.StringFlag{Name: "operator-set", Required: true},
							&cli.Int64Flag{Name: "magnitude", Required: true},
						},
						Action: strategyAllocateCmd,
					},
					{
						Name:  "deallocate",
						Usage: "Deallocate magnitude from an operator set",
						Flags: []cli.Flag{
							&cli.StringFlag{Name: "operator-id", Required: true},
							&cli.StringFlag{Name: "strategy", Required: true},
							&cli.StringFlag{Name: "operator-set", Required: true},
							&cli.Int64Flag{Name: "magnitude", Required: true},
						},
						Action: strategyDeallocateCmd,
					},
					{
						Name:  "deposit",
						Usage: "Simulate staker deposit to a strategy",
						Flags: []cli.Flag{
							&cli.StringFlag{Name: "operator-id", Required: true},
							&cli.StringFlag{Name: "strategy", Required: true},
							&cli.Int64Flag{Name: "amount", Required: true},
						},
						Action: strategyDepositCmd,
					},
				},
			},
		},
	}
}

func operatorListCmd(c *cli.Context) error {
	data, err := operator.LoadOperatorData(operatorDataPath)
	if err != nil {
		return err
	}
	for _, op := range data.Operators {
		fmt.Printf("Operator: %s\n", op.ID)
		for _, sa := range op.Strategies {
			sum := sa.Summary()
			fmt.Printf("  Strategy: %s, Delegated: %d\n", sum.Strategy, sum.Delegated)
			for _, a := range sum.Allocations {
				fmt.Printf("    %s: Magnitude=%d, Proportion=%.2f%%, Amount=%d\n",
					a.OperatorSet, a.Magnitude, a.Proportion*100, a.Amount)
			}
		}
	}
	return nil
}

func strategyListCmd(c *cli.Context) error {
	operatorID := c.String("operator-id")
	data, err := operator.LoadOperatorData(operatorDataPath)
	if err != nil {
		return err
	}
	op, err := data.FindOperator(operatorID)
	if err != nil {
		return err
	}
	for _, sa := range op.Strategies {
		sum := sa.Summary()
		fmt.Printf("Strategy: %s, Delegated: %d\n", sum.Strategy, sum.Delegated)
		for _, a := range sum.Allocations {
			fmt.Printf("  %s: Magnitude=%d, Proportion=%.2f%%, Amount=%d\n",
				a.OperatorSet, a.Magnitude, a.Proportion*100, a.Amount)
		}
	}
	return nil
}

func strategyGetCmd(c *cli.Context) error {
	operatorID := c.String("operator-id")
	strategy := c.String("strategy")
	data, err := operator.LoadOperatorData(operatorDataPath)
	if err != nil {
		return err
	}
	op, err := data.FindOperator(operatorID)
	if err != nil {
		return err
	}
	sa := op.FindStrategy(strategy)
	if sa == nil {
		return fmt.Errorf("strategy %s not found for operator %s", strategy, operatorID)
	}
	sum := sa.Summary()
	fmt.Printf("Strategy: %s, Delegated: %d\n", sum.Strategy, sum.Delegated)
	for _, a := range sum.Allocations {
		fmt.Printf("  %s: Magnitude=%d, Proportion=%.2f%%, Amount=%d\n",
			a.OperatorSet, a.Magnitude, a.Proportion*100, a.Amount)
	}
	return nil
}

func strategyAllocateCmd(c *cli.Context) error {
	operatorID := c.String("operator-id")
	strategy := c.String("strategy")
	operatorSet := c.String("operator-set")
	magnitude := c.Int64("magnitude")
	data, err := operator.LoadOperatorData(operatorDataPath)
	if err != nil {
		return err
	}
	op, err := data.FindOperator(operatorID)
	if err != nil {
		// Create operator if not found
		op = &operator.Operator{ID: operatorID}
		data.Operators = append(data.Operators, *op)
		// Find pointer again (append may reallocate)
		op, _ = data.FindOperator(operatorID)
	}
	sa := op.FindOrCreateStrategy(strategy)
	if err := sa.AllocateMagnitude(operatorSet, magnitude); err != nil {
		return err
	}
	if err := operator.SaveOperatorData(operatorDataPath, data); err != nil {
		return err
	}
	fmt.Println("Allocation successful.")
	return nil
}

func strategyDeallocateCmd(c *cli.Context) error {
	operatorID := c.String("operator-id")
	strategy := c.String("strategy")
	operatorSet := c.String("operator-set")
	magnitude := c.Int64("magnitude")
	data, err := operator.LoadOperatorData(operatorDataPath)
	if err != nil {
		return err
	}
	op, err := data.FindOperator(operatorID)
	if err != nil {
		return err
	}
	sa := op.FindStrategy(strategy)
	if sa == nil {
		return fmt.Errorf("strategy %s not found for operator %s", strategy, operatorID)
	}
	if err := sa.DeallocateMagnitude(operatorSet, magnitude); err != nil {
		return err
	}
	if err := operator.SaveOperatorData(operatorDataPath, data); err != nil {
		return err
	}
	fmt.Println("Deallocation successful.")
	return nil
}

func strategyDepositCmd(c *cli.Context) error {
	operatorID := c.String("operator-id")
	strategy := c.String("strategy")
	amount := c.Int64("amount")
	data, err := operator.LoadOperatorData(operatorDataPath)
	if err != nil {
		return err
	}
	op, err := data.FindOperator(operatorID)
	if err != nil {
		return err
	}
	sa := op.FindStrategy(strategy)
	if sa == nil {
		return fmt.Errorf("strategy %s not found for operator %s", strategy, operatorID)
	}
	if err := sa.Deposit(amount); err != nil {
		return err
	}
	if err := operator.SaveOperatorData(operatorDataPath, data); err != nil {
		return err
	}
	fmt.Println("Deposit successful.")
	return nil
}
