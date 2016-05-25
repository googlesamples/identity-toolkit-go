package favweekday

import (
	"encoding/gob"
	"fmt"
	"html/template"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/google/identity-toolkit-go-client/gitkit"
	"github.com/gorilla/mux"
	"github.com/gorilla/sessions"

	"golang.org/x/net/context"
	"golang.org/x/net/xsrftoken"
	"google.golang.org/appengine"
	"google.golang.org/appengine/datastore"
	aelog "google.golang.org/appengine/log"
	"google.golang.org/appengine/mail"
)

// Templates file path.
const (
	homeTemmplatePath  = "home.tmpl"
	gitkitTemplatePath = "gitkit.tmpl"
)

// Action URLs.
const (
	homeURL          = "/"
	widgetURL        = "/gitkit"
	signOutURL       = "/signOut"
	oobActionURL     = "/oobAction"
	updateURL        = "/update"
	deleteAccountURL = "/deleteAccount"
)

// Identity toolkit configurations.
const (
	browserAPIKey            = "INSERT_YOUR_BROWSER_API_KEY_HERE"
	clientID                 = "INSERT_YOUR_SERVER_CLIENT_ID_HERE"
	googleAppCredentialsPath = "INSERT_YOUR_JSON_KEY_FILE_PATH_HERE"
)

// Cookie/Form input names.
const (
	gtokenCookieName = "gtoken"
	sessionName      = "SESSIONID"
	xsrfTokenName    = "xsrftoken"
	favoriteName     = "favorite"
)

// Email templates.
const (
	emailTemplateResetPassword = `<p>Dear user,</p>
<p>
Forgot your password?<br>
FavWeekday received a request to reset the password for your account <b>%[1]s</b>.<br>
To reset your password, click on the link below (or copy and paste the URL into your browser):<br>
<a href="%[2]s">%[2]s</a><br>
</p>
<p>FavWeekday Support</p>`

	emailTemplateChangeEmail = `<p>Dear user,</p>

<p>
Want to use another email address to sign into FavWeekday?<br>
FavWeekday received a request to change your account email address from %[1]s to <b>%[2]s</b>.<br>
To change your account email address, click on the link below (or copy and paste the URL into your browser):<br>
<a href="%[3]s">%[3]s</a><br>
</p>
<p>FavWeekday Support</p>`

	emailTemplateVerifyEmail = `Dear user,

<p>Thank you for creating an account on FavWeekday.</p>
<p>To verify your account email address, click on the link below (or copy and paste the URL into your browser):</p>
<p><a href="%[1]s">%[1]s</a></p>

<br>
<p>FavWeekday Support</p>`
)

var (
	homeTemplate   = template.Must(template.ParseFiles(homeTemmplatePath))
	gitkitTemplate = template.Must(template.ParseFiles(gitkitTemplatePath))

	weekdays = []time.Weekday{
		time.Sunday,
		time.Monday,
		time.Tuesday,
		time.Wednesday,
		time.Thursday,
		time.Friday,
		time.Saturday,
	}

	xsrfKey      string
	cookieStore  *sessions.CookieStore
	gitkitClient *gitkit.Client
)

// User information.
type User struct {
	ID            string
	Email         string
	Name          string
	EmailVerified bool
}

type SessionUserKey int

// Key used to store the user information in the current session.
const sessionUserKey SessionUserKey = 0

// currentUser extracts the user information stored in current session.
//
// If there is no existing session, identity toolkit token is checked. If the
// token is valid, a new session is created.
//
// If any error happens, nil is returned.
func currentUser(r *http.Request) *User {
	c := appengine.NewContext(r)
	s, _ := cookieStore.Get(r, sessionName)
	if s.IsNew {
		// Extract the token string from request.
		ts := gitkitClient.TokenFromRequest(r)
		if ts == "" {
			return nil
		}
		// Check the token issue time. Only accept token that is no more than 15
		// minitues old even if it's still valid.
		token, err := gitkitClient.ValidateToken(c, ts, []string{clientID})
		if err != nil {
			aelog.Errorf(c, "Invalid token %s: %s", ts, err)
			return nil
		}
		if time.Now().Sub(token.IssueAt) > 15*time.Minute {
			aelog.Infof(c, "Token %s is too old. Issused at: %s", ts, token.IssueAt)
			return nil
		}
		// Fetch user info.
		u, err := gitkitClient.UserByLocalID(c, token.LocalID)
		if err != nil {
			aelog.Errorf(c, "Failed to fetch user info for %s[%s]: %s", token.Email, token.LocalID, err)
			return nil
		}
		return &User{
			ID:            u.LocalID,
			Email:         u.Email,
			Name:          u.DisplayName,
			EmailVerified: u.EmailVerified,
		}
	} else {
		// Extracts user from current session.
		v, ok := s.Values[sessionUserKey]
		if !ok {
			aelog.Errorf(c, "no user found in current session")
		}
		return v.(*User)
	}
}

// saveCurrentUser stores the user information in current session.
func saveCurrentUser(r *http.Request, w http.ResponseWriter, u *User) {
	if u == nil {
		return
	}
	s, _ := cookieStore.Get(r, sessionName)
	s.Values[sessionUserKey] = *u
	err := s.Save(r, w)
	if err != nil {
		aelog.Errorf(appengine.NewContext(r), "Cannot save session: %s", err)
	}
}

type FavWeekday struct {
	// User ID. Serves as primary key in datastore.
	ID string
	// 0 is Sunday.
	Weekday time.Weekday
}

// weekdayForUser fetches the favorite weekday for the user from the datastore.
// Sunday is returned if no such data is found.
func weekdayForUser(r *http.Request, u *User) time.Weekday {
	c := appengine.NewContext(r)
	k := datastore.NewKey(c, "FavWeekday", u.ID, 0, nil)
	d := FavWeekday{}
	err := datastore.Get(c, k, &d)
	if err != nil {
		if err != datastore.ErrNoSuchEntity {
			aelog.Errorf(c, "Failed to fetch the favorite weekday for user %+v: %s", *u, err)
		}
		return time.Sunday
	}
	return d.Weekday
}

// updateWeekdayForUser updates the favorite weekday for the user.
func updateWeekdayForUser(r *http.Request, u *User, d time.Weekday) {
	c := appengine.NewContext(r)
	k := datastore.NewKey(c, "FavWeekday", u.ID, 0, nil)
	_, err := datastore.Put(c, k, &FavWeekday{u.ID, d})
	if err != nil {
		aelog.Errorf(c, "Failed to update the favorite weekday for user %+v: %s", *u, err)
	}
}

func handleHome(w http.ResponseWriter, r *http.Request) {
	u := currentUser(r)
	var d time.Weekday
	if u != nil {
		d = weekdayForUser(r, u)
	}
	saveCurrentUser(r, w, u)
	var xf, xd string
	if u != nil {
		xf = xsrftoken.Generate(xsrfKey, u.ID, updateURL)
		xd = xsrftoken.Generate(xsrfKey, u.ID, deleteAccountURL)
	}
	homeTemplate.Execute(
		w,
		struct {
			WidgetURL              string
			SignOutURL             string
			User                   *User
			WeekdayIndex           int
			Weekdays               []time.Weekday
			UpdateWeekdayURL       string
			UpdateWeekdayXSRFToken string
			DeleteAccountURL       string
			DeleteAccountXSRFToken string
		}{widgetURL, signOutURL, u, int(d), weekdays, updateURL, xf, deleteAccountURL, xd})
}

func handleWidget(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()
	// Extract the POST body if any.
	b, _ := ioutil.ReadAll(r.Body)
	body, _ := url.QueryUnescape(string(b))
	gitkitTemplate.Execute(
		w,
		struct {
			BrowserAPIKey    string
			SignInSuccessUrl string
			OOBActionURL     string
			POSTBody         string
		}{browserAPIKey, homeURL, oobActionURL, body})
}

func handleSignOut(w http.ResponseWriter, r *http.Request) {
	s, _ := cookieStore.Get(r, sessionName)
	s.Options = &sessions.Options{
		MaxAge: -1, // MaxAge<0 means delete session cookie.
	}
	err := s.Save(r, w)
	if err != nil {
		aelog.Errorf(appengine.NewContext(r), "Cannot save session: %s", err)
	}
	// Also clear identity toolkit token.
	http.SetCookie(w, &http.Cookie{Name: gtokenCookieName, MaxAge: -1})
	// Redirect to home page for sign in again.
	http.Redirect(w, r, homeURL, http.StatusFound)
}

func handleOOBAction(w http.ResponseWriter, r *http.Request) {
	c := appengine.NewContext(r)
	resp, err := gitkitClient.GenerateOOBCode(c, r)
	if err != nil {
		aelog.Errorf(c, "Failed to get an OOB code: %s", err)
		w.Write([]byte(gitkit.ErrorResponse(err)))
		return
	}
	msg := &mail.Message{
		Sender: "FavWeekday Support <support@favweekday.appspot.com>",
	}
	switch resp.Action {
	case gitkit.OOBActionResetPassword:
		msg.Subject = "Reset your FavWeekday account password"
		msg.HTMLBody = fmt.Sprintf(emailTemplateResetPassword, resp.Email, resp.OOBCodeURL.String())
		msg.To = []string{resp.Email}
	case gitkit.OOBActionChangeEmail:
		msg.Subject = "FavWeekday account email address change confirmation"
		msg.HTMLBody = fmt.Sprintf(emailTemplateChangeEmail, resp.Email, resp.NewEmail, resp.OOBCodeURL.String())
		msg.To = []string{resp.NewEmail}
	case gitkit.OOBActionVerifyEmail:
		msg.Subject = "FavWeekday account registration confirmation"
		msg.HTMLBody = fmt.Sprintf(emailTemplateVerifyEmail, resp.OOBCodeURL.String())
		msg.To = []string{resp.Email}
	}
	if err := mail.Send(c, msg); err != nil {
		aelog.Errorf(c, "Failed to send %s message to user %s: %s", resp.Action, resp.Email, err)
		w.Write([]byte(gitkit.ErrorResponse(err)))
		return
	}
	w.Write([]byte(gitkit.SuccessResponse()))
}

func handleUpdate(w http.ResponseWriter, r *http.Request) {
	c := appengine.NewContext(r)
	var (
		d   int
		day time.Weekday
		err error
	)
	// Check if there is a signed in user.
	u := currentUser(r)
	if u == nil {
		aelog.Errorf(c, "No signed in user for updating")
		goto out
	}
	// Validate XSRF token first.
	if !xsrftoken.Valid(r.PostFormValue(xsrfTokenName), xsrfKey, u.ID, updateURL) {
		aelog.Errorf(c, "XSRF token validation failed")
		goto out
	}
	// Extract the new favorite weekday.
	d, err = strconv.Atoi(r.PostFormValue(favoriteName))
	if err != nil {
		aelog.Errorf(c, "Failed to extract new favoriate weekday: %s", err)
		goto out
	}
	day = time.Weekday(d)
	if day < time.Sunday || day > time.Saturday {
		aelog.Errorf(c, "Got wrong value for favorite weekday: %d", d)
	}
	// Update the favorite weekday.
	updateWeekdayForUser(r, u, day)
out:
	// Redirect to home page to show the update result.
	http.Redirect(w, r, homeURL, http.StatusFound)
}

func handleDeleteAccount(w http.ResponseWriter, r *http.Request) {
	c := appengine.NewContext(r)
	// Check if there is a signed in user.
	u := currentUser(r)
	if u == nil {
		aelog.Errorf(c, "No signed in user for updating")
		goto out
	}
	// Validate XSRF token first.
	if !xsrftoken.Valid(r.PostFormValue(xsrfTokenName), xsrfKey, u.ID, deleteAccountURL) {
		aelog.Errorf(c, "XSRF token validation failed")
		goto out
	}
	// Delete account.
	if err := gitkitClient.DeleteUser(c, &gitkit.User{LocalID: u.ID}); err != nil {
		aelog.Errorf(c, "Failed to delete user %+v: %s", *u, err)
		goto out
	}
	// Account deletion succeeded. Call sign out to clear session and identity
	// toolkit token.
	handleSignOut(w, r)
	return
out:
	http.Redirect(w, r, homeURL, http.StatusFound)
}

func init() {
	// Register datatypes such that it can be saved in the session.
	gob.Register(SessionUserKey(0))
	gob.Register(&User{})

	// Initialize XSRF token key.
	xsrfKey = "My very secure XSRF token key"

	// Create a session cookie store.
	cookieStore = sessions.NewCookieStore(
		[]byte("My very secure authentication key for cookie store or generate one using securecookies.GenerateRamdonKey()")[:64],
		[]byte("My very secure encryption key for cookie store or generate one using securecookies.GenerateRamdonKey()")[:32])
	cookieStore.Options = &sessions.Options{
		MaxAge:   86400 * 7, // Session valid for one week.
		HttpOnly: true,
	}

	// Create identity toolkit client.
	c := &gitkit.Config{
		WidgetURL: widgetURL,
	}
	if appengine.IsDevAppServer() {
		c.GoogleAppCredentialsPath = googleAppCredentialsPath
	}
	var err error
	gitkitClient, err = gitkit.New(context.Background(), c)
	if err != nil {
		log.Fatal(err)
	}

	r := mux.NewRouter()
	r.HandleFunc(homeURL, handleHome)
	r.HandleFunc(widgetURL, handleWidget)
	r.HandleFunc(signOutURL, handleSignOut)
	r.HandleFunc(oobActionURL, handleOOBAction)
	r.HandleFunc(updateURL, handleUpdate)
	r.HandleFunc(deleteAccountURL, handleDeleteAccount)
	http.Handle("/", r)
}
