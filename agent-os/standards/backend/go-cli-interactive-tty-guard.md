# Go CLI Interactive TTY Guard

- Interactive CLIs require both stdin and stdout to be TTY before launching UI
- If not TTY: print stub output, then `Interactive terminal required to launch UI.`; exit `0`

```go
if !term.IsTerminal(int(os.Stdin.Fd())) || !term.IsTerminal(int(os.Stdout.Fd())) {
  _ = scaffold.WriteStubMessage(os.Stdout, bin)
  _, _ = fmt.Fprintln(os.Stdout, "Interactive terminal required to launch UI.")
  return
}
```
