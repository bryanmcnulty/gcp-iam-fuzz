package cmd

import (
  "context"
  "encoding/json"
  "errors"
  "fmt"
  "gcp-iam-fuzz/pkg/data"
  "gcp-iam-fuzz/pkg/iamfuzz"
  "github.com/rs/zerolog"
  "github.com/spf13/cobra"
  "os"
  "sync"
)

var (
  argDebug   bool
  argJson    bool
  argLogJson bool
  argTasks   int
  argProject string
  argToken   string
  argOutput  string
)

var rootCmd = &cobra.Command{
  Use:   "gcp-iam-fuzz",
  Short: "Quickly enumerate IAM permissions for a GCP account",

  Args: func(cmd *cobra.Command, args []string) error {
    if argTasks < 0 || argTasks > 100 {
      return errors.New("tasks must be between 1 and 100")
    }
    if argProject == "" {
      return errors.New("project ID is required")
    } else if argToken == "" {
      return errors.New("GCP access token is required")
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

    groupSize := len(data.AllPerms) / argTasks
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

    for i := range argTasks {
      groups[i] = &iamfuzz.Task{
        In:  data.AllPerms[i*groupSize : (i*groupSize)+groupSize],
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
        log.Error().Msg("Failed to write output as JSON")
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
  rootCmd.Flags().BoolVarP(&argDebug, "debug", "d", false, "Enable debug logging")
  rootCmd.Flags().BoolVarP(&argJson, "json", "j", false, "Enable JSON output")
  rootCmd.Flags().BoolVarP(&argLogJson, "log-json", "l", false, "Log messages in JSON format")
  rootCmd.Flags().IntVarP(&argTasks, "threads", "T", 10, "Number of concurrent threads")
  rootCmd.Flags().StringVarP(&argProject, "project", "p", "", "GCP project name")
  rootCmd.Flags().StringVarP(&argToken, "token", "t", "", "GCP access token")
  rootCmd.Flags().StringVarP(&argOutput, "output", "o", "", "Output file path")
}

func Execute() {
  if err := rootCmd.Execute(); err != nil {
    fmt.Println(err)
    os.Exit(1)
  }
}
