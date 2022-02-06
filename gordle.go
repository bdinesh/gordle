package main

import (
	"bufio"
	"embed"
	"flag"
	"fmt"
	"io/fs"
	"math/rand"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"
)

const (
	totalWordles = 2315
	blackSquare  = `⬛`
	yellowSquare = `🟨`
	greenSquare  = `🟩`
)

var (
	//go:embed guesses.txt
	//go:embed answers.txt
	files embed.FS

	dayFlag    = flag.Int("day", daysSinceFirstWordle(), "Select a specific wordle by day")
	randomFlag = flag.Bool("random", false, "pick a random wordle")

	firstDay = time.Date(2021, time.June, 19, 0, 0, 0, 0, time.UTC)
	valid    = regexp.MustCompile(`^[A-Za-z]{5}$`)
)

type cell struct {
	color  string
	letter string
}

type game struct {
	day            int
	currentTurn    int
	turnsRemaining int
	complete       bool
	won            bool
	answer         string
	validGuesses   map[string]struct{}
	board          [][]cell
}

func main() {
	flag.Parse()
	rand.Seed(time.Now().UnixNano())

	var day int

	if *randomFlag {
		day = randomDay()
	} else {
		day = *dayFlag
	}

	g := newGame(day)
	s := bufio.NewScanner(os.Stdin)

	g.printTurn()

	for !g.complete && s.Scan() {
		guess := strings.ToUpper(strings.TrimSpace(s.Text()))

		if !valid.MatchString(guess) {
			g.printTurnWithError("Please enter a 5 letter word")
			continue
		}

		if _, ok := g.validGuesses[guess]; !ok {
			g.printTurnWithError("Not in word list")
			continue
		}

		g.addGuess(guess)
		g.printTurn()
	}

	if s.Err() != nil {
		panic(s.Err())
	}

	// prevents answer from printing if user used a signal to end the program
	// scanner.Err() returns nil if io.EOF
	if g.complete {
		g.printShareableScore()
	}
}

func (g *game) printShareableScore() {
	var turnS string

	if g.won {
		turnS = strconv.Itoa(g.currentTurn)
		fmt.Println("You won!")
	} else {
		turnS = "X"
		fmt.Println("You lose!")
		fmt.Println("Answer was", g.answer)
	}

	fmt.Printf("Wordle %v %v/6\n\n", g.day, turnS)

	for i := 0; i < g.currentTurn; i++ {
		for _, x := range g.board[i] {
			fmt.Print(x.color)
		}
		fmt.Println()
	}
}

func green(l string) string {
	return "\033[37;102m" + l + "\033[0m"
}

func yellow(l string) string {
	return "\033[37;103m" + l + "\033[0m"
}

//func white(l string) string {
//	return "\033[0;107m" + l + "\033[0m"
//}

func black(l string) string {
	return "\033[37;100m" + l + "\033[0m"
}

func clearBoard() {
	fmt.Print("\033c")
}

func prompt() {
	fmt.Print(">")
}

func newGame(day int) *game {
	board := make([][]cell, 6)

	for i := range board {
		board[i] = make([]cell, 5)

		for j := range board[i] {
			board[i][j] = cell{
				color:  blackSquare,
				letter: "_",
			}
		}
	}

	return &game{
		day:            day,
		currentTurn:    0,
		turnsRemaining: 6,
		complete:       false,
		won:            false,
		answer:         answerForDay(day),
		validGuesses:   guessesSet(),
		board:          board,
	}
}

func (g *game) addGuess(guess string) {
	// A cell is green if the letters by index match.
	// A cell is yellow if
	//   the cell is not green
	//   the letter exists somewhere in the word
	//   the sum of green cells and yellow cells for the letter is less than the frequency of the letter
	// Otherwise, the cell is black.
	freq := make(map[rune]int)

	for _, r := range g.answer {
		freq[r]++
	}

	// A cell is green if the letters by index match.
	for i, r := range guess {
		if rune(g.answer[i]) == r {
			freq[r]--
			g.board[g.currentTurn][i].color = greenSquare
		}

		g.board[g.currentTurn][i].letter = string(r)
	}

	for i, r := range guess {
		// not green, it exists somewhere, and there is room left to color
		if rune(g.answer[i]) != r && strings.ContainsRune(g.answer, r) && freq[r] > 0 {
			freq[r]--
			g.board[g.currentTurn][i].color = yellowSquare
			g.board[g.currentTurn][i].letter = string(r)
		}
	}

	g.turnsRemaining--
	g.currentTurn++

	if guess == g.answer || g.turnsRemaining == 0 {
		g.complete = true
	}

	if guess == g.answer {
		g.won = true
	}
}

func (g *game) print() {
	fmt.Printf("Wordle %v\n", g.day)

	for _, row := range g.board {
		for _, cell := range row {
			l := cell.letter

			switch cell.color {
			case greenSquare:
				l = green(l)
			case yellowSquare:
				l = yellow(l)
			default:
				l = black(l)
			}

			fmt.Print(" " + l)
		}

		fmt.Println()
	}
}

func (g *game) printTurn() {
	clearBoard()
	g.print()
	prompt()
}

func (g *game) printTurnWithError(err string) {
	clearBoard()
	g.print()
	fmt.Println(err)
	prompt()
}

func closeFile(f fs.File) {
	_ = f.Close()
}

func guessesSet() map[string]struct{} {
	guessesFile, err := files.Open("guesses.txt")
	if err != nil {
		panic(err)
	}

	defer closeFile(guessesFile)

	validGuesses := make(map[string]struct{})
	guessesReader := bufio.NewScanner(guessesFile)

	for guessesReader.Scan() {
		validGuesses[strings.ToUpper(guessesReader.Text())] = struct{}{}
	}

	if guessesReader.Err() != nil {
		panic(err)
	}

	return validGuesses
}

func answerForDay(day int) string {
	answersFile, err := files.Open("answers.txt")
	if err != nil {
		panic(err)
	}

	defer closeFile(answersFile)

	scanner := bufio.NewScanner(answersFile)

	for i := 0; i <= day; i++ {
		scanner.Scan()
	}

	return strings.ToUpper(scanner.Text())
}

func daysSinceFirstWordle() int {
	year, month, day := time.Now().Date()
	today := time.Date(year, month, day, 0, 0, 0, 0, time.UTC)
	return int(today.Sub(firstDay).Hours() / 24)
}

func randomDay() int {
	return rand.Intn(totalWordles)
}
