## Cedana Testing

Integration tests are found in `integration`. Otherwise unit tests are found in this folder for components that _require_ unit tests. Please don't make unit tests for the sake of it.

### Dev scripts

On a macbook, it can be difficult to test some of the checkpointing code without launching a full blown vm. Unfortunately orbstack doesn't really cut it here either, since they use a modified kernel.

Good thing we have a way to run instances and load them up with stuff we want to run anywhere :)
To spin up a dev environment, simply call `cedana-cli run dev.yml`, after changing the requisite parameters in the yaml file (such as the branch you want cloned).

You can then call `cedana-cli ssh instanceID` to get the parameters needed to pass through to your editor. If using VSCode (as the default example shows):

- Click on the green bottom-left square (or view the Remote Explorer tab), which represents the Remote - SSH extension.
- Click on the + symbol to add a new SSH host.
- Enter the SSH command to connect to your remote instance, it will look like ssh -i key.pem user@hostname -p port.
- Select a configuration file to save this setting, usually it's located at ~/.ssh/config.
- You'll see the new SSH host in the SSH Targets section. Click on the folder icon to the right of it to connect.
- VSCode will open a new window and you'll be connected to your remote instance.
