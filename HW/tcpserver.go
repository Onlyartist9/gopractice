package main

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"log"
	"net"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode"
)

type serverclientkeypair struct {
	ClientID int
	ServerID int
}

type Point struct {
	x int64
	y int64
}

//Maneuvering maps
var turningleft = map[Point]Point{
	North:   West,
	West:    South,
	South:   East,
	East:    North,
	Point{}: Point{},
}

var turningright = map[Point]Point{
	North:   East,
	West:    North,
	South:   West,
	East:    South,
	Point{}: Point{},
}

var orientationsinwords = map[Point]string{
	North:       "North",
	East:        "East",
	West:        "West",
	South:       "South",
	Point{0, 0}: "Undefined",
}

type Robot struct {
	previouslocation Point
	currentlocation  Point
	orientation      Point
}

func (arobot Robot) evade(someconnection net.Conn) Robot {

	fmt.Printf("\nEvading...\n\n")
	arobot = arobot.turnright(someconnection)
	arobot.moveforward(someconnection)
	arobot = arobot.turnleft(someconnection)
	arobot.moveforward(someconnection)
	arobot = arobot.turnleft(someconnection)
	fmt.Printf("\nDone evading.\n")

	return arobot
}

func (arobot *Robot) turnright(someconnection net.Conn) Robot {
	fmt.Printf("Moving right\n")
	_, err := io.WriteString(someconnection, serverturnright)

	if err != nil {
		log.Print(err)
	}

	scanner := bufio.NewScanner(someconnection)
	scanner.Split(ScanLines)
	scanner.Scan()
	buff1 := scanner.Bytes()
	fmt.Printf("Location returned during right turn was %s\n", buff1)

	arobot.orientation = turningright[arobot.orientation]
	fmt.Printf("Our robot's current orientation is %s\n", orientationsinwords[arobot.orientation])
	return *arobot
}

func (arobot *Robot) turnleft(someconnection net.Conn) Robot {
	fmt.Printf("Moving left\n")
	_, err := io.WriteString(someconnection, serverturnleft)

	if err != nil {
		log.Print(err)
	}

	scanner := bufio.NewScanner(someconnection)
	scanner.Split(ScanLines)
	scanner.Scan()
	buff1 := scanner.Bytes()
	fmt.Printf("Location returned during left turn was %s\n", buff1)

	arobot.orientation = turningleft[arobot.orientation]
	fmt.Printf("Our robot's current orientation is %s\n", orientationsinwords[arobot.orientation])
	return *arobot
}

func (arobot *Robot) moveforward(someconnection net.Conn) {
	fmt.Printf("Moving\n")
	_, err := io.WriteString(someconnection, servermove)

	if err != nil {
		log.Print(err)
	}

	scanner := bufio.NewScanner(someconnection)
	scanner.Split(ScanLines)
	scanner.Scan()
	buff1 := scanner.Bytes()
	arobot.previouslocation = arobot.currentlocation
	arobot.currentlocation.x, arobot.currentlocation.y, _ = getcoordinates(buff1, arobot.currentlocation.x, arobot.currentlocation.y)
	*arobot = CalculateOrientation(*arobot)
	fmt.Printf("Location x = %d, and location y = %d\n", arobot.currentlocation.x, arobot.currentlocation.y)

	return
}

func EqualPoints(somepointa Point, somepointb Point) bool {
	if somepointa.x == somepointb.x && somepointa.y == somepointb.y {
		fmt.Printf("The positions/directions are equal\n")
		return true
	}
	return false
}

func CalculateOrientation(robot Robot) Robot {
	robot.orientation = Point{robot.currentlocation.x - robot.previouslocation.x, robot.currentlocation.y - robot.previouslocation.y}
	fmt.Printf("Our new orientation is directionx = %d, and directiony = %d\n", robot.orientation.x, robot.orientation.y)
	return robot
}

//Orientations
var North Point = Point{0, 1}
var East Point = Point{1, 0}
var West Point = Point{-1, 0}
var South Point = Point{0, -1}
var Zeropoint = Point{0, 0}

//Constants
const (
	maxlengthforclientname    = 18
	maxlengthforkeyid         = 5
	timeout                   = 2 * time.Second
	maxclientconfirmlength    = 7
	maxclientoklength         = 12
	maxclientrecharginglength = 12
	maxclientfullpowerlength  = 12
	maxclientmessagelength    = 100
	suffix                    = "\a\b"
	serverKeyRequest          = "107 KEY REQUEST\a\b"
	serverOK                  = "200 OK\a\b"
	serverloginfailed         = "300 LOGIN FAILED\a\b"
	serverok                  = "200 OK\a\b"
	servermove                = "102 MOVE\a\b"
	serverturnleft            = "103 TURN LEFT\a\b"
	serverturnright           = "104 TURN RIGHT\a\b"
	serverpickupmessage       = "105 GET MESSAGE\a\b"
	serverlogout              = "106 LOGOUT\a\b"
	serversyntaxerror         = "301 SYNTAX ERROR\a\b"
	serverlogicerror          = "302 LOGIC ERROR\a\b"
	serverkeyoutofrange       = "303 KEY OUT OF RANGE\a\b"
	clientrecharging          = "RECHARGING\a\b"
	clientfullpower           = "FULL POWER\a\b"
	thespot                   = "OK 0 0\a\b"
	delimiter                 = byte('\b')
	keyrelatederror           = "Key related error"
	controlrelatederror       = "Control related error"
	successful                = "Suscessful logout"
)

func main() {
	listener, err := net.Listen("tcp", "localhost:8000")
	if err != nil {
		log.Fatal(err)
	}
	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Print(err)
		}
		go handleConn(conn)
	}

}

func handleConn(someconnection net.Conn) {
	//Handle authentication
	command, scanner := authentication(someconnection)

	if command == serverloginfailed {
		fmt.Printf("Server login has failed\n")
		_, err := io.WriteString(someconnection, serverloginfailed)

		if err != nil {
			log.Print(err)
		}

		someconnection.Close()
		return

	} else if command == keyrelatederror {

		fmt.Printf("Key was wrong and so closing connection\n")
		someconnection.Close()

		return

	} else if command == serversyntaxerror {

		_, err := io.WriteString(someconnection, serversyntaxerror)

		if err != nil {
			log.Print(err)
		}

		someconnection.Close()
		return

	} else {
		_, err := io.WriteString(someconnection, serverOK)
		if err != nil {
			log.Print(err)
		}
	}

	//Initialize our local robot
	var Dobbytherobot Robot

	//Start the control of our robot
	command = control(someconnection, scanner, Dobbytherobot)
	if command == controlrelatederror {
		fmt.Printf("Something went wrong moving our robot\n")
		someconnection.Close()
		return
	}
	if command == serversyntaxerror {
		fmt.Printf("Something went wrong moving our robot. Syntax related\n")
		_, err := io.WriteString(someconnection, serversyntaxerror)

		if err != nil {
			log.Print(err)
		}

		someconnection.Close()
		return
	} else if command == successful {
		fmt.Printf("Successfully logged out\n")
		someconnection.Close()
		return
	}

}

func authentication(someconnection net.Conn) (string, bufio.Scanner) {
	//Definining map that stores keypair values
	var idzero serverclientkeypair
	var idone serverclientkeypair
	var idtwo serverclientkeypair
	var idthree serverclientkeypair
	var idfour serverclientkeypair

	idzero.ClientID = 32037
	idone.ClientID = 29295
	idtwo.ClientID = 13603
	idthree.ClientID = 29533
	idfour.ClientID = 21952

	idzero.ServerID = 23019
	idone.ServerID = 32037
	idtwo.ServerID = 18789
	idthree.ServerID = 16443
	idfour.ServerID = 18189

	keyIDtable := map[string]serverclientkeypair{
		"0": idzero,
		"1": idone,
		"2": idtwo,
		"3": idthree,
		"4": idfour,
	}

	//Acquire the username
	username, error, scanner := getusername(someconnection)
	if error == serversyntaxerror {
		return serversyntaxerror, scanner
	}

	//Send request for client key
	requestkey(someconnection)

	//Acquire key
	id, error, scanner := getkeyID(someconnection, scanner)
	if error == serversyntaxerror || error == serverkeyoutofrange {
		return keyrelatederror, scanner
	}

	//Declare server and client keys
	serverkeyID := keyIDtable[strconv.Itoa(id)].ServerID
	clientkeyID := keyIDtable[strconv.Itoa(id)].ClientID

	//Send Server Confirmation
	hash := sendserverconfirm(id, serverkeyID, someconnection, username)

	//Receive Client confirmation
	hash2, hashedconfirm, errormessage, scanner := getclientconfirm(someconnection, clientkeyID, hash, scanner)
	if errormessage == serversyntaxerror {
		return serversyntaxerror, scanner
	}

	fmt.Printf("Our hash 2 is %d, and our hashedconfirm is %d", hash2, hashedconfirm)

	//Check if login is permitted
	command := login(someconnection, hash2, hashedconfirm)
	return command, scanner
}

func control(someconnection net.Conn, scanner bufio.Scanner, arobot Robot) string {
	//Variables to monitor where we are
	arobot.currentlocation = Point{0, 0}
	arobot.previouslocation = Point{0, 0}
	arobot.orientation = Point{0, 0}
	var errorstring string

	//Identify our position
	arobot, errorstring = identifyposition(someconnection, arobot, scanner)
	if errorstring == serversyntaxerror {
		return errorstring
	}

	fmt.Printf("Our starting x position is %v, and our starting y position is %v \n", arobot.currentlocation.x, arobot.currentlocation.y)

	//Move towards target position
	errorstring = guidetodestination(someconnection, arobot)
	if errorstring == serversyntaxerror {
		return controlrelatederror
	}

	//Get message
	errormessage := getmessage(someconnection)
	if errormessage == serversyntaxerror {
		return serversyntaxerror
	}

	successful := logout(someconnection)

	return successful
}

func getusername(someconnection net.Conn) (string, string, bufio.Scanner) {
	var errormessage string
	var username string

	//Get the username
	scanner := bufio.NewScanner(someconnection)
	scanner.Split(ScanLines)
	scanner.Scan()
	username = scanner.Text()
	err := scanner.Err()

	if err != nil {
		fmt.Printf("There's a  %s error", err)
	}

	fmt.Printf("\nThe length of the username is: %d\n", len(username))

	if len(username) >= maxlengthforclientname {
		// There's an error in the username's length
		fmt.Printf("The username is in the wrong format\n")
		errormessage = serversyntaxerror
	}

	fmt.Printf("The username is: %s\n", username)
	byteform := []byte(username)
	fmt.Printf("The byte form of the username is :\n")
	fmt.Println(byteform)

	return username, errormessage, *scanner
}

func requestkey(someconnection net.Conn) {
	fmt.Printf("Sending Key Request...\n")
	_, err := io.WriteString(someconnection, serverKeyRequest)
	if err != nil {
		fmt.Print("We probably closed the connection\n")
	}
}

func getkeyID(someconnection net.Conn, scanner bufio.Scanner) (int, string, bufio.Scanner) {
	//Hash
	var error string
	scanner.Scan()
	buff1 := scanner.Bytes()
	err := scanner.Err()
	if err != nil {
		fmt.Printf("There's a  %s error", err)
	}

	id, err := strconv.Atoi(strings.Trim(string(buff1), "\b"))
	fmt.Printf("Our id is %s\n", buff1)

	if err != nil {
		fmt.Printf("The key %s is not a number\n", buff1)
		_, err = io.WriteString(someconnection, serversyntaxerror)
		error = serversyntaxerror

	}

	if id > 4 || id < 0 || unicode.IsNumber(rune(id)) {
		fmt.Printf("The key %d is out of range\n", id)
		_, err = io.WriteString(someconnection, serverkeyoutofrange)
		error = serverkeyoutofrange
	}

	return id, error, scanner
}

func sendserverconfirm(id int, serverkeyID int, someconnection net.Conn, username string) int {
	fmt.Printf("\n\nThe key id is: %d\n\n", id)
	fmt.Printf("Processing client key...\n")
	fmt.Printf("\n\nServer key to be hashed is: %d\n", serverkeyID)

	//Calculate hash
	fmt.Printf("\n\nCalculating Hash...\n\n")

	hash := 0
	for i, r := range username {
		fmt.Printf("\nCurrently summing index %d and value %q with integer value %d.\n", i, r, r)
		hash += int(r)
	}

	hash *= 1000
	hash1 := hash % 65536
	hash1 = (hash + serverkeyID) % 65536

	wordedhash := fmt.Sprint(hash1)

	fmt.Printf("\nOur hashed value is %d and the message to be sent is %s.\n", hash1, wordedhash+suffix)

	_, err := io.WriteString(someconnection, fmt.Sprint(hash1)+suffix)
	if err != nil {
		log.Fatal(err)
	}
	return hash
}

func getclientconfirm(someconnection net.Conn, clientkeyID int, hash int, scanner bufio.Scanner) (int, int, string, bufio.Scanner) {
	var errormessage string
	var hash2 int
	var hashedconfirm int
	fmt.Printf("Setting deadline\n")
	err := someconnection.SetDeadline(time.Now().Add(timeout))
	if err != nil {
		fmt.Printf("Error occured setting deadline\n")
	}
	fmt.Printf("Setting timeout\n")

	scanner.Scan()
	if scanner.Err() != nil {
		timeouterror := scanner.Err().(net.Error).Timeout()
		fmt.Printf("Error should show here\n")
		if timeouterror {
			errormessage = serversyntaxerror
			return hash2, hashedconfirm, errormessage, scanner
		}
	}
	err = someconnection.SetDeadline(time.Time{})
	buff1 := scanner.Bytes()
	fmt.Printf("\nComparing hashes...\n")

	confirmationmessage := string(buff1)
	fmt.Printf("Our confirmation message is %s\n", confirmationmessage)
	hash2 = (hash + clientkeyID) % 65536

	confirmationmessage = strings.Trim(confirmationmessage, "\a\b\x00")

	fmt.Printf("Our confirmation message without prefix is %s\n\n", confirmationmessage)

	hashedconfirm, err = strconv.Atoi(confirmationmessage)

	if hashedconfirm > 99999 {
		fmt.Printf("The hash confirm %d is larger than acceptable \n", hashedconfirm)
		errormessage = serversyntaxerror
	}

	if err != nil {
		fmt.Printf("Problem converting the string to an integer")
		errormessage = serversyntaxerror
	}
	return hash2, hashedconfirm, errormessage, scanner
}

func login(someconnection net.Conn, hash2 int, hashedconfirm int) string {
	if hash2 == hashedconfirm {
		fmt.Printf("\nHashes are equal. Sending confirmation...\n")
		return serverOK

	} else {
		fmt.Printf("\nHashes are not equal. Sending notice of login failure...\n")
		return serverloginfailed
	}

}

func identifyposition(someconnection net.Conn, arobot Robot, scanner bufio.Scanner) (Robot, string) {
	fmt.Printf("Identifying location...\n")
	var errorstring string

	//Move position once and await reply
	fmt.Printf("Moving...\n")
	_, err := io.WriteString(someconnection, servermove)
	if err != nil {
		log.Fatal(err)
	}

	//See if the coordinate syntax is right
	scanner.Scan()
	buff1 := scanner.Bytes()
	location := strings.TrimPrefix(string(buff1)+suffix, "\b")
	validmove := regexp.MustCompile(`OK [-0-9]+ [-0-9]+` + suffix)

	//Identify the result of the first move
	fmt.Printf("The initial location sent is %v\n", location)
	if !validmove.MatchString(location) {
		fmt.Printf("Coordinates are in the wrong format\n")
		errorstring = serversyntaxerror
		return arobot, errorstring
	} else {
		fmt.Printf("Coordinates are in the right format\n")
	}

	//If already at 0,0 we return
	if location == thespot {
		fmt.Printf("Already there\n")
		arobot.currentlocation = Point{0, 0}
		return arobot, errorstring
	}

	//Else assign our initial positions
	arobot.previouslocation.x, arobot.previouslocation.y, errorstring = getcoordinates(buff1, arobot.previouslocation.x, arobot.previouslocation.y)
	if errorstring == serversyntaxerror {
		return arobot, errorstring
	}
	fmt.Printf("Our initial positions are x=%d and y=%d\n", arobot.previouslocation.x, arobot.previouslocation.y)

	//Move position again and await reply
	fmt.Printf("Moving again...\n")
	_, err = io.WriteString(someconnection, servermove)
	if err != nil {
		log.Fatal(err)
	}
	scanner.Scan()
	buff1 = scanner.Bytes()
	fmt.Printf("The next pair of coordinate sent is %s\n", buff1)

	//If we move and find that we are on the spot, we return
	if string(buff1)+suffix == thespot {
		fmt.Printf("Already there\n")
		arobot.currentlocation = Point{0, 0}
		return arobot, errorstring
	}

	//Else we initialize the next pair of coordinates
	arobot.currentlocation.x, arobot.currentlocation.y, errorstring = getcoordinates(buff1, arobot.currentlocation.x, arobot.currentlocation.y)

	//If we haven't changed positions(there's an obstacle)
	fmt.Printf("Checking if initial and current locations are same\n")
	if EqualPoints(arobot.currentlocation, arobot.previouslocation) {
		//evade
		arobot = arobot.evade(someconnection)
	}

	//Determine the orientation
	arobot = CalculateOrientation(arobot)

	fmt.Printf("Currently at x=%d, currently at y=%d\n", arobot.currentlocation.x, arobot.currentlocation.y)

	return arobot, errorstring

}
func guidetodestination(someconnection net.Conn, arobot Robot) string {
	var errorstring string

	for {
		//Top Right
		if arobot.currentlocation.x >= 0 && arobot.currentlocation.y > 0 {
			fmt.Printf("In top right quadrant\n")
			for arobot.orientation != South {
				arobot = arobot.turnright(someconnection)
			}

			//Top-left or x-axis to the left
		} else if arobot.currentlocation.x < 0 && arobot.currentlocation.y >= 0 {
			fmt.Printf("In top left quadrant or x-axis to the left\n")
			for arobot.orientation != East {
				arobot = arobot.turnright(someconnection)
			}

		} else if arobot.currentlocation.x <= 0 && arobot.currentlocation.y < 0 {
			fmt.Printf("In lower left quadrant or \n")
			for arobot.orientation != North {
				arobot = arobot.turnright(someconnection)
			}

		} else if arobot.currentlocation.x > 0 && arobot.currentlocation.y <= 0 {
			fmt.Printf("Lower right quadrant or along right part of x axist\n")
			for arobot.orientation != West {
				arobot = arobot.turnright(someconnection)
			}

		}
		//Move
		arobot.moveforward(someconnection)
		if errorstring == serversyntaxerror {
			return errorstring
		}
		//Calculate new direction
		arobot = CalculateOrientation(arobot)

		fmt.Printf("Currently at x = %d, y = %d,\n", arobot.currentlocation.x, arobot.currentlocation.y)

		if arobot.currentlocation == Zeropoint {
			fmt.Printf("On 'The Spot'.\n")
			return errorstring
		}

		if arobot.currentlocation == arobot.previouslocation {
			fmt.Printf("Hit an obstacle\n")
			arobot = arobot.evade(someconnection)
		}
	}
}

func getmessage(someconnection net.Conn) string {
	var errorstring string
	fmt.Printf("Picking up message\n")

	_, err := io.WriteString(someconnection, serverpickupmessage)

	if err != nil {
		fmt.Printf("Couldn't send request to client")
	}

	buff1 := make([]byte, maxclientmessagelength)
	someconnection.Read(buff1)
	message := strings.Trim(string(buff1), "\x00")
	if len(message) >= 100 {
		return serversyntaxerror
	}
	fmt.Printf("%s\n", message)

	return errorstring
}

func logout(someconnection net.Conn) string {
	fmt.Printf("Logging out\n")
	_, err := io.WriteString(someconnection, serverlogout)
	if err != nil {
		fmt.Printf("Connection probably closed or there's an error writing to buff")
	}
	return successful
}

func getcoordinates(buff1 []byte, x int64, y int64) (int64, int64, string) {

	var location string
	var errorstring string
	location = strings.Trim(string(buff1), "\x00")
	location = strings.TrimSuffix(location, "\a\b")
	location = strings.TrimPrefix(location, "\b")
	location = strings.TrimPrefix(location, "OK")
	fmt.Printf("The location is:%s\n", location)

	coordinatesscanner := bufio.NewScanner(strings.NewReader(location))
	coordinatesscanner.Split(bufio.ScanWords)
	fmt.Printf("Our buffer shows coordinates :%s for coordinate scanning\n", location)
	coordinatesscanner.Scan()
	x, err := strconv.ParseInt(coordinatesscanner.Text(), 10, 64)

	if err != nil {
		fmt.Printf("\nSome probablem associated with reading x coordinate occured\n")
		errorstring = serversyntaxerror
	}

	coordinatesscanner.Scan()

	y, err = strconv.ParseInt(coordinatesscanner.Text(), 10, 64)

	if err != nil {
		fmt.Printf("\nSome probablem associated with reading y coordinate occured\n")
		errorstring = serversyntaxerror
	}

	if err := coordinatesscanner.Err(); err != nil {
		fmt.Printf("Error occured during reading\n")
		errorstring = serversyntaxerror
	}

	return x, y, errorstring

}

func ScanLines(data []byte, atEOF bool) (advance int, token []byte, err error) {
	if atEOF && len(data) == 0 {
		return 0, nil, nil
	}

	if i := bytes.Index(data, []byte{'\a', '\b'}); i >= 0 {
		// We have a full newline-terminated line.
		return i + 1, dropend(data[0:i]), nil
	}

	if atEOF {
		return len(data), dropend(data), nil
	}
	// Request more data.
	return 0, nil, nil
}

func dropend(data []byte) []byte {
	if len(data) > 0 && data[len(data)-1] == '\b' {
		return data[0 : len(data)-1]
	}
	return data
}
