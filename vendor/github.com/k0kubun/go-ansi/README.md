# go-ansi

Windows-portable ANSI escape sequence utility for Go language

## What's this?

This library converts ANSI escape sequences to Windows API calls on Windows environment.  
You can easily use this feature by replacing `fmt` with `ansi`.

![](http://i.gyazo.com/12ecc4e1b4387f5c56d3e6ae319ab6c4.png)
![](http://i.gyazo.com/c41072712ee05e28565ca92b416675e2.png)

### Output redirection

Many coloring libraries for Go just use ANSI escape sequences, which don't work on Windows.

- [fatih/color](https://github.com/fatih/color)
- [mitchellh/colorstring](https://github.com/mitchellh/colorstring)

If you use go-ansi, you can use these libraries' nice APIs for Windows too.
Not only coloring, many ANSI escape sequences are available.

```go
color.Output = ansi.NewAnsiStdout()
color.Cyan("fatih/color")

colorstring.Fprintln(ansi.NewAnsiStdout(), "[green]mitchellh/colorstring")
```

### Cursor

You can control cursor in your terminal. Of course it works on cmd.exe.
In a following table, "Shell" shows a unix-like shortcut for the action.
(It is not provided by this library and just for the explanation.)

| API | Escape Code | Shell | Description |
|:----|:----------------|:--|:------------|
| ansi.CursorUp(n) | CSI `n` A | C-p | Move the cursor n cells to up |
| ansi.CursorDown(n) | CSI `n` B | C-n | Move the cursor n cells to down |
| ansi.CursorForward(n) | CSI `n` C | C-f | Move the cursor n cells to right |
| ansi.CursorBack(n) | CSI `n` D | C-b | Move the cursor n cells to left |
| ansi.CursorNextLine(n) | CSI `n` E | C-n C-a | Move cursor to beginning of the line n lines down. |
| ansi.CursorPreviousLine(n) | CSI `n` F | C-p C-a | Move cursor to beginning of the line n lines up. |
| ansi.CursorHorizontalAbsolute(x) | CSI `n` G | C-a,<br>C-e | Moves the cursor to column n. |

### Display

You can easily control your terminal display. You can easily provide unix-like
shell functionarities for display, such as C-k or C-l.

| API | Escape Code | Shell | Description |
|:----|:----------------|:--|:------------|
| ansi.EraseInLine(n) | CSI `n` K | C-k, C-u,<br>C-a C-k | 0: clear to the end of the line. <br> 1: clear to the beginning of the line. <br> 2: clear entire line. |

## API document

https://godoc.org/github.com/k0kubun/go-ansi

## Notes

This is just a cursor and display supported version of [mattn/go-colorable](https://github.com/mattn/go-colorable).
I used almost the same implementation as it for coloring. Many thanks for [@mattn](https://github.com/mattn).

## License

MIT License
