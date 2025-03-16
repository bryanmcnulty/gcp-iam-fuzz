package iamfuzz

import (
  "bytes"
  "context"
  "encoding/json"
  "fmt"
  "github.com/rs/zerolog"
  "io"
  "net/http"
  "regexp"
)

var (
  InvalidPermissionRegex = regexp.MustCompile(`Permission (.+) is not valid for this resource\.$`)
)

func EnumPerms(ctx context.Context, token, project string, task *Task) {

  log := zerolog.Ctx(ctx).With().Str("project", project).Logger()
  url := fmt.Sprintf("https://cloudresourcemanager.googleapis.com/v1/projects/%s:testIamPermissions", project)
  tryPerms := task.In

  for len(tryPerms) != 0 {

    select {
    case <-ctx.Done():
      log.Warn().Msg("routine cancelled")
      return

    default:
      reqPerms := tryPerms[:]
      if len(tryPerms) > 100 {
        reqPerms = tryPerms[:100]
      }
      requestBody, err := json.Marshal(map[string][]string{
        "permissions": reqPerms,
      })
      if err != nil {
        log.Error().Err(err).Msg("Failed to marshal JSON request body")
        task.Err <- fmt.Errorf("marshal request body: %w", err)
        return
      }
      bodyBuffer := bytes.NewBuffer(requestBody)
      request, err := http.NewRequest(http.MethodPost, url, bodyBuffer)
      if err != nil {
        log.Error().Err(err).Msg("Failed to build HTTP request")
        task.Err <- fmt.Errorf("build request: %w", err)
        return
      }
      request.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))
      request.Header.Set("Content-Type", "application/json")

      log.Debug().Bytes("body", requestBody).Msg("Sending HTTP request")
      resp, err := http.DefaultClient.Do(request)
      if err != nil {
        log.Error().Err(err).Str("url", url).Msg("HTTP POST request failed")
        task.Err <- fmt.Errorf("http post: %w", err)
        return
      }
      responseBody, err := io.ReadAll(resp.Body)
      if err != nil {
        log.Error().Err(err).Msg("Failed to read HTTP response body")
        task.Err <- fmt.Errorf("read response: %w", err)
        return
      }
      if err = resp.Body.Close(); err != nil {
        log.Error().Err(err).Msg("Failed to close response body")
      }

      log.Debug().Bytes("body", responseBody).Msg("Got response")
      ser := make(map[string]any)

      if err = json.Unmarshal(responseBody, &ser); err != nil {
        log.Error().Err(err).Msg("Failed to unmarshal HTTP response from JSON")
        task.Err <- fmt.Errorf("unmarshal http response: %w", err)
        return
      }
      if errorValue, ok := ser["error"]; !ok {
        if permsAny, ok := ser["permissions"]; ok {
          if permsArr, ok := permsAny.([]any); ok {

            log.Info().Any("permission", permsArr).Msg("Found granted permission(s)")

            for _, permAny := range permsArr {
              if perm, ok := permAny.(string); ok {
                task.Out <- perm
              }
            }
          }
        }
        tryPerms = tryPerms[len(reqPerms):]
      } else if errDict, ok := errorValue.(map[string]any); ok {

        if statusAny, ok := errDict["status"]; !ok {
          log.Debug().Msg("Could not determine error status")
        } else if status, ok := statusAny.(string); !ok {
          log.Debug().Msg("Invalid error status returned")

        } else if msgAny, ok := errDict["message"]; !ok {
          log.Debug().Msg("Could not determine message type")
        } else if msg, ok := msgAny.(string); !ok {
          log.Debug().Msg("Invalid message value returned")

        } else {
          log.Debug().Str("status", status).Str("detail", msg).Msg("Got error")

          if match := InvalidPermissionRegex.FindStringSubmatch(msg); len(match) > 1 {
            log.Debug().Str("permission", match[1]).Msg("Invalid permission detected")
            for i, perm := range tryPerms {
              if perm == match[1] {
                newValid := tryPerms[0:i]

                if len(newValid) != 0 {
                  log.Debug().Strs("permissions", newValid).Msg("Valid permission(s) detected")
                }
                tryPerms = append(newValid, tryPerms[i+1:]...)
              }
            }
          } else {
            task.Err <- fmt.Errorf("error: %s - %s", status, msg)
            ctx.Done()
            return
          }
        }
      }
    }
  }
}
