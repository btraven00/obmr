package cmd

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/btraven00/obflow/internal/config"
	"github.com/spf13/cobra"
)

func newConfigCmd() *cobra.Command {
	var unset bool
	var list bool
	c := &cobra.Command{
		Use:   "config [key] [value]",
		Short: "Get or set a config value (git-config style)",
		Long: `Read or write a single key in ./.obflow/config.yaml.

Supported keys:
  default.plan
  omnibenchmark.version
  omnibenchmark.branch
  omnibenchmark.pr

Examples:
  obflow config                                # list all
  obflow config omnibenchmark.branch           # print value
  obflow config omnibenchmark.branch dev       # set value
  obflow config --unset omnibenchmark.branch   # clear`,
		Args: cobra.MaximumNArgs(2),
		RunE: func(_ *cobra.Command, args []string) error {
			cwd, err := os.Getwd()
			if err != nil {
				return err
			}
			cp := config.Find(cwd)
			var cfg *config.Config
			if cp != "" {
				cfg, err = config.Load(cp)
				if err != nil {
					return err
				}
			}
			if cfg == nil {
				cfg = &config.Config{}
			}

			if list || (len(args) == 0 && !unset) {
				printConfig(cfg)
				return nil
			}
			if len(args) < 1 {
				return fmt.Errorf("key required")
			}
			key := args[0]
			if unset {
				if err := setKey(cfg, key, ""); err != nil {
					return err
				}
			} else if len(args) == 1 {
				v, err := getKey(cfg, key)
				if err != nil {
					return err
				}
				fmt.Println(v)
				return nil
			} else {
				if err := setKey(cfg, key, args[1]); err != nil {
					return err
				}
			}
			out, err := config.Save(cwd, cfg)
			if err != nil {
				return err
			}
			fmt.Printf("wrote %s\n", out)
			return nil
		},
	}
	c.Flags().BoolVar(&unset, "unset", false, "clear the named key")
	c.Flags().BoolVar(&list, "list", false, "print all config values")
	return c
}

func printConfig(c *config.Config) {
	if c.Default.Plan != "" {
		fmt.Printf("default.plan = %s\n", c.Default.Plan)
	}
	if c.Omnibenchmark.Version != "" {
		fmt.Printf("omnibenchmark.version = %s\n", c.Omnibenchmark.Version)
	}
	if c.Omnibenchmark.Branch != "" {
		fmt.Printf("omnibenchmark.branch = %s\n", c.Omnibenchmark.Branch)
	}
	if c.Omnibenchmark.PR != 0 {
		fmt.Printf("omnibenchmark.pr = %d\n", c.Omnibenchmark.PR)
	}
}

func getKey(c *config.Config, key string) (string, error) {
	switch strings.ToLower(key) {
	case "default.plan":
		return c.Default.Plan, nil
	case "omnibenchmark.version":
		return c.Omnibenchmark.Version, nil
	case "omnibenchmark.branch":
		return c.Omnibenchmark.Branch, nil
	case "omnibenchmark.pr":
		if c.Omnibenchmark.PR == 0 {
			return "", nil
		}
		return strconv.Itoa(c.Omnibenchmark.PR), nil
	default:
		return "", fmt.Errorf("unknown key: %s", key)
	}
}

func setKey(c *config.Config, key, value string) error {
	switch strings.ToLower(key) {
	case "default.plan":
		c.Default.Plan = value
	case "omnibenchmark.version":
		c.Omnibenchmark.Version = value
	case "omnibenchmark.branch":
		c.Omnibenchmark.Branch = value
	case "omnibenchmark.pr":
		if value == "" {
			c.Omnibenchmark.PR = 0
		} else {
			n, err := strconv.Atoi(value)
			if err != nil {
				return fmt.Errorf("omnibenchmark.pr must be an integer: %w", err)
			}
			c.Omnibenchmark.PR = n
		}
	default:
		return fmt.Errorf("unknown key: %s", key)
	}
	return nil
}
