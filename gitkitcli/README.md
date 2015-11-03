This is an example program that demonstrating how to use gitkit.Client. It is
also helpful for site admins to make API calls to identitytoolkit sevices to
get/set user account information.

To obtain a JSON key file for the service account, you need to go Google Cloud
Console and create one service account if you don't have one. Choose JSON when
downloading the key.

There are two configurations used by this tool:
- ClientID: the OAuth2 client ID for the web server.
- GoogleAppCredentialsPath: path to the JSON key file of the service account. if
  it's absent, Google Application Default is used.

You can provide a JSON configuration file, e.g., config.json:
```
{
  "clientId": "123.apps.googleusercontent.com",
  "GoogleAppCredentialsPath": "/path/to/json/key/file"
}
```

To run the program with the configuration, e.g., get account information for
user "user@example.com":
```
gitkitcli -config_file=config.json getuser user@example.com
```

You can also overwrite a configuration by passing its value from the
correspoding flag:
```
gitkitcli -config_file=config.json -google_app_credentials_path=/path/to/json/key/file createuser
```

If no configuration file is provided through the flag, an environment variable
`GITKIT_CONFIG_FILE` is checked. If it is set, its value is used as the
configuration file path.

For all supported command, run
```
gitkitcli help
```
For help information of a specific command, run
```
gitkitcli help getuser
```
