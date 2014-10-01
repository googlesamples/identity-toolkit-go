// Copyright 2014 Google Inc. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha1"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math/big"
	"net/mail"
	"os"

	"github.com/codegangsta/cli"
	"github.com/google/identity-toolkit-go-client/gitkit"
	"github.com/howeyc/gopass"
)

func main() {
	app := cli.NewApp()
	app.Name = "gitkitcli"
	app.Usage = "command line tool for Google Identity Toolkit service"
	app.Version = "0.1"
	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name: "config_file",
			Usage: "the JSON format configuration file." +
				"The value in the config file could be overwritten by the corresponding config flag.",
			EnvVar: "GITKIT_CONFIG_FILE",
		},
		cli.StringFlag{
			Name:  "client_id",
			Usage: "the client ID of the web server.",
		},
		cli.StringFlag{
			Name:  "service_account",
			Usage: "the service account email address.",
		},
		cli.StringFlag{
			Name:  "key_path",
			Usage: "the PEM encoding private key file path for the service account.",
		},
	}
	app.Before = initClient
	app.Commands = []cli.Command{
		commandValidateToken(),
		commandGetUser(),
		commandUpdateUser(),
		commandDeleteUser(),
		commandCreateUser(),
		commandUploadUsers(),
		commandDownloadUsers(),
	}
	app.RunAndExitOnError()
}

var client *gitkit.Client

func initClient(c *cli.Context) error {
	configFile := c.String("config_file")
	var config *gitkit.Config
	var err error
	if configFile != "" {
		config, err = gitkit.LoadConfig(configFile)
		if err != nil {
			return err
		}
	} else {
		config = &gitkit.Config{}
	}
	// It is required but not used.
	config.WidgetURL = "http://localhost"
	// Command line flags overwrite the values in config file.
	if c.IsSet("client_id") {
		config.ClientID = c.String("client_id")
	}
	if c.IsSet("service_account") {
		config.ServiceAccount = c.String("service_account")
	}
	if c.IsSet("key_path") {
		config.PEMKeyPath = c.String("key_path")
	}

	if client, err = gitkit.New(config, nil); err != nil {
		return err
	}
	return nil
}

func checkZeroArgument(c *cli.Context) error {
	if n := len(c.Args()); n != 0 {
		return fmt.Errorf("except 0 argument but got %d", n)
	}
	return nil
}

func checkOneArgument(c *cli.Context) error {
	if n := len(c.Args()); n != 1 {
		return fmt.Errorf("except 1 argument but got %d", n)
	}
	return nil
}

func checkZeroOrOneArgument(c *cli.Context) error {
	if n := len(c.Args()); n > 1 {
		return fmt.Errorf("except 0 or 1 argument but got %d", n)
	}
	return nil
}

func printUser(user *gitkit.User) {
	b, err := json.MarshalIndent(user, "", "  ")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(string(b))
}

func failOnError(c *cli.Context, err error) {
	if err != nil {
		log.Fatalf("Fail to execute command %s: %s", c.Command.Name, err)
	}
}

// getUserByIdentifier retrieves the account information specified by the
// identifier, which could be an email addresss, a local ID or an ID token.
func getUserByIdentifier(identifier string) (*gitkit.User, error) {
	if _, err := mail.ParseAddress(identifier); err == nil {
		return client.UserByEmail(identifier)
	} else if _, err := client.ValidateToken(identifier); err == nil {
		return client.UserByToken(identifier)
	} else {
		return client.UserByLocalID(identifier)
	}
}

func generateUser(email, password string, key, salt []byte) (*gitkit.User, error) {
	u := gitkit.User{Email: email, Salt: salt}
	mac := hmac.New(sha1.New, key)
	mac.Write([]byte(password))
	mac.Write(salt)
	u.PasswordHash = mac.Sum(nil)
	r, err := rand.Int(rand.Reader, big.NewInt(1e16))
	if err != nil {
		return nil, err
	}
	u.LocalID = fmt.Sprintf("id%16d", r)
	return &u, nil
}

// readUsers reads the next n users from the decoder stream.
func readUsers(d *json.Decoder, n int) ([]*gitkit.User, error) {
	var users []*gitkit.User
	for i := 0; i < n; i++ {
		var u gitkit.User
		if err := d.Decode(&u); err != nil {
			return users, err
		}
		users = append(users, &u)
	}
	return users, nil
}

func commandValidateToken() cli.Command {
	return cli.Command{
		Name:        "validatetoken",
		Usage:       "validatetoken ID_TOKEN",
		Description: "Validate the given ID token and print the account information contained in it.",
		Action: func(c *cli.Context) {
			failOnError(c, checkOneArgument(c))
			u, err := client.ValidateToken(c.Args().First())
			failOnError(c, err)
			fmt.Println(">> token info:")
			printUser(u)
		},
	}
}

func commandGetUser() cli.Command {
	return cli.Command{
		Name:        "getuser",
		Usage:       "getuser EMAIL|LOCAL_ID|ID_TOKEN",
		Description: "Get the account information of the user specified by the email address, local user ID or ID token.",
		Action: func(c *cli.Context) {
			failOnError(c, checkOneArgument(c))
			u, err := getUserByIdentifier(c.Args().First())
			failOnError(c, err)
			fmt.Println(">> user info:")
			printUser(u)
		},
	}
}

func commandUpdateUser() cli.Command {
	return cli.Command{
		Name:        "updateuser",
		Usage:       "updateuser [Options] EMAIL|LOCAL_ID|ID_TOKEN",
		Description: "Update the account information of the user specified by the email address, local user ID or ID token.",
		Flags: []cli.Flag{
			cli.StringFlag{
				Name:  "name",
				Usage: "the new display name for the user.",
			},
			cli.BoolFlag{
				Name:  "password",
				Usage: "whether to set a new password.",
			},
			cli.BoolFlag{
				Name:  "email_verified",
				Usage: "whether the email address is verified.",
			},
		},
		Action: func(c *cli.Context) {
			failOnError(c, checkOneArgument(c))
			u, err := getUserByIdentifier(c.Args().First())
			failOnError(c, err)
			if c.IsSet("name") {
				u.DisplayName = c.String("name")
			}
			if c.IsSet("password") {
				fmt.Print("New password: ")
				password := string(gopass.GetPasswd())
				if password != "" {
					u.Password = password
				}
			}
			if c.IsSet("email_verified") {
				u.EmailVerified = c.Bool("email_verified")
			}
			failOnError(c, client.UpdateUser(u))
			if c.IsSet("password") {
				// If a new password is set, the new PasswordHash need to be retrieved.
				if u, err = getUserByIdentifier(u.LocalID); err != nil {
					failOnError(c, err)
				}
			}
			fmt.Println(">> user updated:")
			printUser(u)
		},
	}
}

func commandDeleteUser() cli.Command {
	return cli.Command{
		Name:        "deleteuser",
		Usage:       "deleteuser EMAIL|LOCAL_ID|ID_TOKEN",
		Description: "Delete a user specified by the email address, local user ID or ID token.",
		Action: func(c *cli.Context) {
			failOnError(c, checkOneArgument(c))
			u, err := getUserByIdentifier(c.Args().First())
			failOnError(c, err)
			failOnError(c, client.DeleteUser(u))
			fmt.Println(">> user deleted:")
			printUser(u)
		},
	}
}

func commandCreateUser() cli.Command {
	return cli.Command{
		Name:        "createuser",
		Usage:       "createuser",
		Description: "Create a new user account. The email address and password are prompted to enter.",
		Action: func(c *cli.Context) {
			failOnError(c, checkZeroArgument(c))
			key := make([]byte, 32)
			salt := make([]byte, 10)
			if _, err := rand.Read(key); err != nil {
				failOnError(c, err)
			}
			if _, err := rand.Read(salt); err != nil {
				failOnError(c, err)
			}
			fmt.Print("Email: ")
			var email string
			fmt.Scanf("%s\n", &email)
			fmt.Print("Password: ")
			password := string(gopass.GetPasswd())
			u, err := generateUser(email, password, key, salt)
			failOnError(c, err)
			failOnError(c, client.UploadUsers([]*gitkit.User{u}, "HMAC_SHA1", key, nil))
			u, err = getUserByIdentifier(u.Email)
			failOnError(c, err)
			fmt.Println(">> user created:")
			printUser(u)
		},
	}
}

func commandUploadUsers() cli.Command {
	return cli.Command{
		Name:        "uploadusers",
		Usage:       "uploadusers [Options] USERS_FILE",
		Description: "Upload the user accounts in the file.",
		Flags: []cli.Flag{
			cli.StringFlag{
				Name:  "algorithm",
				Usage: "the algorithm name for hashing password.",
			},
			cli.StringFlag{
				Name:  "hash_key",
				Usage: "URL safe base64 encoded hash key.",
			},
			cli.StringFlag{
				Name:  "salt_separator",
				Usage: "URL safe base64 encoded salt separator.",
			},
		},
		Action: func(c *cli.Context) {
			failOnError(c, checkOneArgument(c))
			if !c.IsSet("algorithm") || c.IsSet("hash_key") {
				failOnError(c, fmt.Errorf("-algorithm and -hash_key are required"))
			}
			key, err := base64.URLEncoding.DecodeString(c.String("hash_key"))
			failOnError(c, err)
			separator, err := base64.URLEncoding.DecodeString(c.String("salt_separator"))
			failOnError(c, err)
			f, err := os.Open(c.Args().First())
			failOnError(c, err)
			defer f.Close()
			d := json.NewDecoder(f)
			for done := false; !done; {
				users, err := readUsers(d, 20)
				if err == io.EOF {
					done = true
				} else {
					failOnError(c, err)
				}
				err = client.UploadUsers(users, c.String("algorithm"), key, separator)
				if uploadErr, ok := err.(gitkit.UploadError); ok {
					for _, v := range uploadErr {
						fmt.Printf(">> failed to upload user %s: %s\n", users[v.Index].Email, v.Message)
					}
				} else {
					failOnError(c, err)
				}
			}
			fmt.Println(">> done")
		},
	}
}

func commandDownloadUsers() cli.Command {
	return cli.Command{
		Name:        "downloadusers",
		Usage:       "downloadusers [output]",
		Description: "Download all user accounts. If output is not specified or -, standard output is used.",
		Action: func(c *cli.Context) {
			failOnError(c, checkZeroOrOneArgument(c))
			var f *os.File
			var err error
			if len(c.Args()) == 0 || c.Args().First() == "-" {
				f = os.Stdout
			} else {
				f, err = os.OpenFile(c.Args().First(), os.O_WRONLY|os.O_TRUNC|os.O_CREATE, os.FileMode(0600))
				failOnError(c, err)
				defer f.Close()
			}
			l := client.ListUsers()
			maxRetries := 5
			i := 0
			for {
				for u := range l.C {
					b, err := json.MarshalIndent(u, "", "  ")
					failOnError(c, err)
					fmt.Fprintln(f, string(b))
				}
				if l.Error != nil && i < maxRetries {
					i++
					l.Retry()
					continue
				}
				break
			}
			failOnError(c, l.Error)
			fmt.Println(">> done")
		},
	}
}
