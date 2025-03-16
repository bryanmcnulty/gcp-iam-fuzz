package cmd

import (
  "context"
  "encoding/json"
  "errors"
  "fmt"
  "github.com/bryanmcnulty/gcp-iam-fuzz/pkg/data"
  "github.com/bryanmcnulty/gcp-iam-fuzz/pkg/iamfuzz"
  "github.com/rs/zerolog"
  "github.com/spf13/cobra"
  "os"
  "sync"
)

var (
  argTasks   int = 6
  argDebug   bool
  argJson    bool
  argLogJson bool
  argProject string
  argToken   string
  argOutput  string
)

var rootCmd = &cobra.Command{
  Use:   "gcp-iam-fuzz",
  Short: "Quickly enumerate IAM permissions for a GCP account",
  Long: `gcp-iam-fuzz is a tool to quickly enumerate IAM permissions for a GCP account

Author: Bryan McNulty (@bryanmcnulty)
Source: https://github.com/bryanmcnulty/gcp-iam-fuzz
`,

  Args: func(cmd *cobra.Command, args []string) error {
    if argTasks < 0 || argTasks > 100 {
      return errors.New("tasks must be between 1 and 100")
    }
    if argProject == "" {
      return errors.New("project ID (-p / --project) is required")
    }
    if argToken == "" {
      if argToken = os.Getenv("GCP_ACCESS_TOKEN"); argToken == "" {
        return errors.New("access token (-t / --token / GCP_ACCESS_TOKEN) is required")
      }
    }
    return cobra.NoArgs(cmd, args)
  },

  RunE: func(cmd *cobra.Command, _ []string) (err error) {
    var results []string
    var ctx context.Context
    var log zerolog.Logger
    var of *os.File

    if argOutput != "" {
      if of, err = os.OpenFile(argOutput, os.O_WRONLY|os.O_CREATE, 0644); err != nil {
        return fmt.Errorf("failed to open output file: %w", err)
      }
    } else {
      of = os.Stdout
    }

    if argDebug {
      zerolog.SetGlobalLevel(zerolog.DebugLevel)
    } else {
      zerolog.SetGlobalLevel(zerolog.InfoLevel)
    }
    if argLogJson {
      log = zerolog.New(os.Stderr).With().Timestamp().Logger()
    } else {
      log = zerolog.New(zerolog.ConsoleWriter{Out: os.Stderr}).With().Timestamp().Logger()
    }

    permsSize := len(data.AllPerms)
    groupSize := permsSize / argTasks

    groups := make([]*iamfuzz.Task, argTasks)
    out := make(chan string)
    errc := make(chan error, argTasks)
    ctx = context.Background()

    wg := &sync.WaitGroup{}
    wg.Add(argTasks)

    go func() {
      for p := range out {
        results = append(results, p)
        if !argJson {
          if _, err = of.WriteString(p + "\n"); err != nil {
            errc <- fmt.Errorf("failed to write output: %w", err)
            ctx.Done()
          }
        }
      }
    }()
    go func() {
      for e := range errc {
        log.Error().Err(e).Msg("Task error")
      }
    }()

    var chunk []string

    for i := range argTasks {
      if i == argTasks-1 {
        chunk = data.AllPerms[i*groupSize:]
      } else {
        chunk = data.AllPerms[i*groupSize : (i*groupSize)+groupSize]
      }
      groups[i] = &iamfuzz.Task{
        In:  chunk,
        Out: out,
        Err: errc,
      }
      go func() {
        defer func() {
          log.Debug().Int("task", i).Msg("Task complete")
          wg.Done()
        }()
        iamfuzz.EnumPerms(log.With().Int("task", i).Logger().WithContext(ctx), argToken, argProject, groups[i])
      }()
    }
    wg.Wait()
    if argJson {
      var content []byte

      if content, err = json.MarshalIndent(map[string][]string{"permissions": results}, "", "  "); err != nil {
        log.Error().Msg("Failed to serialize output to JSON")
        content = []byte("{}\n")
      }
      if _, err = of.Write(content); err != nil {
        log.Error().Err(err).Msg("Failed to write output")
      }
    }
    return nil
  },
}

func init() {
  rootCmd.Flags().BoolVarP(&argDebug, "debug", "d", argDebug, "Enable debug logging")
  rootCmd.Flags().BoolVarP(&argJson, "json", "j", argJson, "Enable JSON output")
  rootCmd.Flags().BoolVarP(&argLogJson, "log-json", "l", argLogJson, "Log messages in JSON format")
  rootCmd.Flags().IntVarP(&argTasks, "threads", "T", argTasks, "Number of concurrent threads")
  rootCmd.Flags().StringVarP(&argProject, "project", "p", argProject, "GCP project name")
  rootCmd.Flags().StringVarP(&argToken, "token", "t", argToken, "GCP access token. environment variable GCP_ACCESS_TOKEN may also be used")
  rootCmd.Flags().StringVarP(&argOutput, "output", "o", argOutput, "Output file path")

  if err := rootCmd.MarkFlagRequired("project"); err != nil {
    panic(err)
  }
}

func Execute() {
  if err := rootCmd.Execute(); err != nil {
    fmt.Println(err)
    os.Exit(1)
  }
}
