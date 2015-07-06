This is an example program that demonstrating how to use gitkit.Client. It is
also helpful for site admins to make API calls to identitytoolkit sevices to
get/set user account information.

To obtain a private key for the service account, you need to go Google Cloud
Console and create one service account if you don't have one. The downloaded
private key is in a PKCS12 encoding file, you can convert it to a PEM encoding
by openssl:
```
$ openssl pkcs12 -in <key.p12> -nocerts -passin pass:notasecret -nodes \
-out <key.pem>
```

There are four required configurations by this tool:
- ClientID: the OAuth2 client ID for the web server.
- ServerAPIKey: the API key for the server to fetch the identitytoolkit public certificates.
- ServiceAccount: the email address of the service account.
- PEMKeyPath: the PEM enconding private key file path.

You can provide a JSON configuration file, e.g., config.json:
```
{
  "clientId": "123.apps.googleusercontent.com",
  "serverApiKey": "server_api_key",
  "serviceAccount": "123-abc@developer.gserviceaccount.com",
  "keyPath": "/dir-of-your-key/private-key.pem"
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
gitkitcli -config-file=config.json -key_path=/new-key-dir/key.pem createuser
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
