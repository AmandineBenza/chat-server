/*

	Nice-Sophia-Antipolis University.
	Master II - IFI - Software Architecture.
	2018-2019.

	Concurrent programming - Chat server in Go.

	BENZA Amandine
	FORNALI Damien

*/

package main;

import (
	"fmt"
	"net"
	"bufio"
	"log"
	"bytes"
	"strings"
	"time"
);

/*

	-- Types used --

*/


/*
	A user has an id and a pseudo.
	We keep its connection in order to be
	able to send him messages.
*/
type user struct {
	id int;
	pseudo string;
	connection *net.Conn;
}

/*
	A session contains a cluster of users.
	We also keep reference to the session listener
	and the last user id.
	The session also has a users number limit.
*/
type session struct {
	serverListener *net.Listener;
	users []user;
	usersPtr int;
	maxUsersAllowed int;
}

func main(){
	/*
		Launch a new server using TCP.
		URL and port are inquired.
		100: max users number allowed.
	*/
	launchServer("tcp", "localhost", "1234", 100);
}

/*

	-- Server --

*/ 

/*
	Launch the server using the protocole and url inquired.
	The users number limit is also inquired.   
*/
func launchServer(protocole, address, port string, maxUsersAllowed int){
	fmt.Println("> Launching server...");
	listener, err := launchListener(protocole, address, port);
	uniqueSession := buildSession(&listener, maxUsersAllowed);
	fmt.Println("> Session built.");
	handleError(err);
	launchProcess(&uniqueSession, &listener, err);
}


// Launch the users listener.
func launchListener(protocole, address, port string) (listener net.Listener, err error){
	var sbuffer bytes.Buffer;
	sbuffer.WriteString(address);
	sbuffer.WriteString(":");
	sbuffer.WriteString(port);
	return net.Listen(protocole, sbuffer.String());
}

// Build a new session.
func buildSession(listener *net.Listener, maxUsersAllowed int) session {
	var sess session;
	sess.serverListener = listener;
	sess.users = make([]user, maxUsersAllowed);
	sess.usersPtr = 0;
	sess.maxUsersAllowed = maxUsersAllowed;
	return sess;
}

// Launch the server main execution.
func launchProcess(sess *session, listener *net.Listener, err error){
	fmt.Println("> Server started.");
	fmt.Println("> Listening to user connections...");

	for {
		if checkSessionFilled(sess) {
			continue;
		}

		conn, err := (*listener).Accept();

		if handleLogError(err){
			continue;
		}

		go processNewUser(sess, listener, &conn);
	}
}

// Check if the session has reached the users limit.
func checkSessionFilled(sess *session) bool {
	return sess.usersPtr >= sess.maxUsersAllowed - 1;
}

/*

	-- User --

*/

// Process a new user. Create it and handle it.
func processNewUser(sess *session, listener *net.Listener, conn *net.Conn) {
	// create user
	newUser := createUser(sess, listener, conn);
	// handle User
	handleUser(sess, newUser);
}

// Create a new user and displays welcome message.
func createUser(sess *session, listener *net.Listener, conn *net.Conn) *user{
	newUser := user{
		pseudo: "null",
		connection: conn,
		id: sess.usersPtr,
	};

	for {
		(*conn).Write([]byte("> Enter pseudo: "));
		reader := bufio.NewReader(*conn);
		message, err := reader.ReadString('\n');

		if handleLogError(err){
			(*conn).Write([]byte("> Error found. Please try again."));
			continue;
		}

		userMessage := filterPseudo(message);

		if userMessage != "" {
			newUser.pseudo = userMessage;
			break;
		}
	}

	sess.users[sess.usersPtr] = newUser;
	sess.usersPtr++;
	
	fmt.Printf("\"%s\" connected.\n", newUser.pseudo);
	broadcastMessageToAll(sess, buildWelcomeMessage(filterPseudo(newUser.pseudo)));

	if checkSessionFilled(sess){
		fmt.Printf("Users limit reached !\n");
	}

	return &newUser;
}

// Handle a user.
func handleUser(sess *session, _user *user){
	reader := bufio.NewReader(*(_user.connection));

	for {
		message, err := reader.ReadString('\n');

		if handleLogError(err){
			processUserExit(sess, _user);
			return;
		}

		broadcastMessage(sess, _user, buildUserMessage(_user, message));
	}

	processUserExit(sess, _user);
}

// Handle user exit.
func processUserExit(sess *session, _user *user){
	fmt.Printf("\"%s\" disconnected.\n", _user.pseudo);
	broadcastMessage(sess, _user, buildByeMessage(_user.pseudo));
	(*_user.connection).Close();
	sess.users[_user.id] = sess.users[sess.usersPtr];
	sess.users[sess.usersPtr - 1] = user{};
	sess.usersPtr = sess.usersPtr - 1;
}

/*

	-- Messages --

*/ 

// Build a welcome message
func buildWelcomeMessage(userPseudo string) string {
	var sbuffer bytes.Buffer;
	sbuffer.WriteString("> Welcome ");
	sbuffer.WriteString(userPseudo);
	sbuffer.WriteString(" !\n");
	return sbuffer.String();
}

// Build a bye message
func buildByeMessage(userPseudo string) string {
	var sbuffer bytes.Buffer;
	sbuffer.WriteString("> See you later ");
	sbuffer.WriteString(userPseudo);
	sbuffer.WriteString(" !\n");
	return sbuffer.String();
}

// Preparer a user message to be sent 
func buildUserMessage(_user *user, message string) string {
	var sbuffer bytes.Buffer;
	sbuffer.WriteString(_user.pseudo);
	sbuffer.WriteString(": ");
	sbuffer.WriteString(filterMessage(message, "\n"));
	sbuffer.WriteString("\n");
	return sbuffer.String();
}

// Build a timeout message
func buildTimeoutMessage(_user *user) string {
	var sbuffer bytes.Buffer;
	sbuffer.WriteString("> ");
	sbuffer.WriteString(_user.pseudo);
	sbuffer.WriteString(" was idle too long and was disconnected.\n");
	return sbuffer.String();
}

// Send a message to all the users of a session except the messageSender
func broadcastMessage(sess *session, messageSender *user, message string){
	for userId := range sess.users {
		if userId != messageSender.id {
			currentUser := sess.users[userId];
			
			if currentUser == (user{}) {
				break;
			}

			(*currentUser.connection).Write([]byte(message));
		}
	}
}

// Send a message to all the users of a session
func broadcastMessageToAll(sess *session, message string){
	for userId := range sess.users {
		currentUser := sess.users[userId];

		if currentUser == (user{}) {
			break;
		}

		(*currentUser.connection).Write([]byte(message));
	}
}

// Removes 'by' occurrences of 'message' content
func filterMessage(message string, by string) string{
	return strings.Replace(message, by, "", -1);
}

// Returns a proper version of the pseudo inquired
func filterPseudo(pseudo string) string {
	fpseudo := strings.Replace(pseudo, "\n", "", -1);
	fpseudo = strings.Replace(fpseudo, " ", "", -1);
	fpseudo = strings.Replace(fpseudo, "(", "", -1);
	fpseudo = strings.Replace(fpseudo, ")", "", -1);
	fpseudo = strings.Replace(fpseudo, "*", "", -1);
	fpseudo = strings.Replace(fpseudo, "&", "", -1);
	fpseudo = strings.Replace(fpseudo, "#", "", -1);	
	fpseudo = strings.Replace(fpseudo, "'", "", -1);
	fpseudo = strings.Replace(fpseudo, "=", "", -1);
	fpseudo = strings.Replace(fpseudo, "_", "", -1);
	return fpseudo;
}

/*

	-- Errors --

*/ 

func handleError(err error){
	if err != nil {
		log.Fatal(err);
	}
}

func handleLogError(err error) bool{
	errorCheck := err != nil;

	if errorCheck {
		log.Println(err);
	}

	return errorCheck;
}