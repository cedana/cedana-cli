## cedana-cli completion powershell

Generate the autocompletion script for powershell

### Synopsis

Generate the autocompletion script for powershell.

To load completions in your current shell session:

	cedana-cli completion powershell | Out-String | Invoke-Expression

To load completions for every new session, add the output of the above command
to your powershell profile.


```
cedana-cli completion powershell [flags]
```

### Options

```
  -h, --help              help for powershell
      --no-descriptions   disable completion descriptions
```

### Options inherited from parent commands

```
      --config string       one-time config JSON string (merge with existing config)
      --config-dir string   custom config directory
```

### SEE ALSO

* [cedana-cli completion](cedana-cli_completion.md)	 - Generate the autocompletion script for the specified shell

###### Auto generated by spf13/cobra on 3-Apr-2025
