# Go CLI Help + Exit Codes

- Help prints to stdout; accepts `-h`, `--help`, `help`; exit `0`
- Usage/arg errors: print message to stderr; exit `2`
- Unexpected/internal errors: print message to stderr; exit `1`

```go
if err := validate(args); err != nil {
  fmt.Fprintln(os.Stderr, err)
  os.Exit(2)
}

if err := run(); err != nil {
  fmt.Fprintf(os.Stderr, "%s failed: %v\n", bin, err)
  os.Exit(1)
}
```
