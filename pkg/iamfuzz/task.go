package iamfuzz

type Task struct {
  Out chan string
  Err chan error
  In  []string
}
