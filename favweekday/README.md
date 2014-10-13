This is a sample App Engine App demonstrating how Google Identity Toolkit is
integrated.

- Follow the instructions at https://cloud.google.com/sdk/#download to install
the Google Cloud SDK and App Engine Go environment if you don't have one.

- Create a project in Google developers console (or use an existing one) and
configure the Google Identity Toolkit API by following the quick-start
instructions.

- Provide the following configuration in favweekday.go
  * browserAPIKey
  * serverAPIKey
  * clientID
  * serviceAccount
  * privateKeyPath

The service account private key needs to be PEM encoded. You can convert the
downloaded `.p12` file to a `.pem` one using `openssl` tool:
```
openssl pkcs12 -in <key.p12> -nocerts -passin pass:notasecret -nodes -out <key.pem>
```
Note that the service account and private key are needed for running the app in
dev appserver. If you deploy it to App Engine, they are not required so that
you can keep the private key in a safe location in your dev environment and
don't need to upload it.

There are also three keys used in this sample app for session cookie
authentication, encryption and XSRF token signing.
They are only for demonstration and not secure. Be sure to use real secure keys
in your prod app.

- Run the setup.sh script, which fetches the libraries this sample app needs.

- Run the sample app (assuming your current working directory is `favweekday`).
```
dev_appserver.py --enable_sendmail=yes .
```

- (Optional) Deploy the sample app. Update the `application` field in `app.yaml`
to use your App Engine app ID and run (assuming your current working directory
is `favweekday`)
```
goapp deploy
```
