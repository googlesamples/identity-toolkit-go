This is a sample App Engine App demonstrating how Google Identity Toolkit is
integrated.

- Follow the instructions at https://cloud.google.com/sdk/#download to install
the Google Cloud SDK and App Engine Go environment if you don't have one.

- Create a project in Google developers console (or use an existing one) and
configure the Google Identity Toolkit API by following the quick-start
instructions.

- Provide the following configuration in favweekday.go
  * browserAPIKey
  * clientID

You'll need a service account and its JSON key file, which can be created and
downloaded from Google cloud console, for running the app in dev appserver.
If you deploy it to App Engine, they are not required so that you can keep the
JSON key file in a safe location in your dev environment and don't need to
upload it.

There are also three keys used in this sample app for session cookie
authentication, encryption and XSRF token signing. They are only for
demonstration and not secure. Be sure to use real secure keys in your prod app.

Run the sample app (assuming your current working directory is `favweekday`).
```
dev_appserver.py --enable_sendmail=yes .
```

- (Optional) Deploy the sample app. Update the `application` field in `app.yaml`
to use your App Engine app ID and run (assuming your current working directory
is `favweekday`)
```
goapp deploy
```
