## fox completion zsh

Generate the autocompletion script for zsh

### Synopsis

Generate the autocompletion script for the zsh shell.

If shell completion is not already enabled in your environment you will need
to enable it.  You can execute the following once:

	echo "autoload -U compinit; compinit" >> ~/.zshrc

To load completions in your current shell session:

	source <(fox completion zsh)

To load completions for every new session, execute once:

#### Linux:

	fox completion zsh > "${fpath[1]}/_fox"

#### macOS:

	fox completion zsh > $(brew --prefix)/share/zsh/site-functions/_fox

You will need to start a new shell for this setup to take effect.


```
fox completion zsh [flags]
```

### Options

```
  -h, --help              help for zsh
      --no-descriptions   disable completion descriptions
```

### Options inherited from parent commands

```
  -a, --app string                 path to directory containing KubeFox App
  -i, --info                       enable info output
  -o, --output string              output format, one of ["json", "yaml"] (default "yaml")
      --registry-address string    address of your container registry
      --registry-token string      access token for your container registry
      --registry-username string   username for your container registry
  -v, --verbose                    enable verbose output
```

### SEE ALSO

* [fox completion](fox_completion.md)	 - Generate the autocompletion script for the specified shell

