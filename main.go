package main

import (
	"bufio"
	"embed"
	"encoding/json"
	"fmt"
	"math/rand"
	"os"
	"strings"
	"time"

	"github.com/nsf/termbox-go"
)

//go:embed stratagems.json
var embeddedFiles embed.FS

// combination represents a combo loaded from JSON.
type combination struct {
	Name     string `json:"name"`
	Sequence string `json:"sequence"`
}

// Arrow holds the ASCII art and the expected termbox key for detection.
type Arrow struct {
	Art string
	Key termbox.Key
}

// Map runes to Arrow objects.
var arrowsMap = map[rune]Arrow{
	'U': {
		Art: "   ██   \n ██████ \n████████\n   ██   \n   ██   ",
		Key: termbox.KeyArrowUp,
	},
	'D': {
		Art: "   ██   \n   ██   \n████████\n ██████ \n   ██   ",
		Key: termbox.KeyArrowDown,
	},
	'L': {
		Art: "    ███   \n  █████   \n██████████\n  █████   \n    ███   ",
		Key: termbox.KeyArrowLeft,
	},
	'R': {
		Art: "   ███    \n   █████  \n██████████\n   █████  \n   ███    ",
		Key: termbox.KeyArrowRight,
	},
}

// loadCombinations attempts to load the combinations from a local file.
// If the local file is not found, it falls back to the embedded JSON.
func loadCombinations(filename string) ([]combination, error) {
	var data []byte
	var err error
	if fileExists(filename) {
		data, err = os.ReadFile(filename)
		if err != nil {
			return nil, err
		}
	} else {
		data, err = embeddedFiles.ReadFile("stratagems.json")
		if err != nil {
			return nil, err
		}
	}
	var combos []combination
	err = json.Unmarshal(data, &combos)
	if err != nil {
		return nil, err
	}
	return combos, nil
}

// fileExists checks if a file exists and is not a directory.
func fileExists(filename string) bool {
	info, err := os.Stat(filename)
	return err == nil && !info.IsDir()
}

func main() {
	rand.Seed(time.Now().UnixNano())

	// Ask for username.
	fmt.Print("Enter your username: ")
	userScanner := bufio.NewScanner(os.Stdin)
	userScanner.Scan()
	username := strings.TrimSpace(userScanner.Text())

	// Show options.
	fmt.Println("Choose an option:")
	fmt.Println("1: JSON Combos (10 random combos from file)")
	fmt.Println("2: Random Combos (10 random sequences of 6 arrows)")
	fmt.Println("3: Timed JSON Combos (30 seconds to finish 10 random combos)")
	fmt.Println("q: Quit")

	scanner := bufio.NewScanner(os.Stdin)
	scanner.Scan()
	input := scanner.Text()

	var score int
	var elapsed float64

	switch input {
	case "1":
		score, elapsed = playJSONCombos(10)
	case "2":
		score, elapsed = playRandomCombos(10)
	case "3":
		score, elapsed = playTimedJSONCombos(10, 30*time.Second)
	case "q", "Q":
		fmt.Println("Exiting...")
		return
	default:
		fmt.Println("Invalid option, please restart the program.")
		return
	}

	fmt.Printf("Congratulations %s! Final Score: %d in %.2f seconds\n", username, score, elapsed)
	waitForExit()
}

func waitForExit() {
	fmt.Println("Press 'Enter' to exit.")
	bufio.NewReader(os.Stdin).ReadBytes('\n')
}

// playJSONCombos processes count random combos from the JSON file (non-timed mode).
// Returns the total score and elapsed time.
func playJSONCombos(count int) (int, float64) {
	startTime := time.Now()
	if err := termbox.Init(); err != nil {
		fmt.Println("Failed to initialize termbox:", err)
		return 0, 0
	}
	defer termbox.Close()

	combos, err := loadCombinations("stratagems.json")
	if err != nil {
		fmt.Printf("Error loading combinations: %s\n", err)
		return 0, 0
	}

	rand.Shuffle(len(combos), func(i, j int) {
		combos[i], combos[j] = combos[j], combos[i]
	})
	if count > len(combos) {
		count = len(combos)
	}

	totalScore := 0
	fmt.Println("JSON Combos Mode: Solve 10 random combos from the file!")
	for i := 0; i < count; i++ {
		combo := combos[i]
		seq := arrowSequenceFromCombination(combo.Sequence)
		completed, _ := processSequence(seq, &totalScore, combo.Name)
		if !completed {
			fmt.Printf("You exited early. Final Score: %d\n", totalScore)
			return totalScore, time.Since(startTime).Seconds()
		}
	}
	return totalScore, time.Since(startTime).Seconds()
}

// playRandomCombos processes count rounds of random sequences (each with 6 arrows).
// Returns the total score and elapsed time.
func playRandomCombos(count int) (int, float64) {
	startTime := time.Now()
	if err := termbox.Init(); err != nil {
		fmt.Println("Failed to initialize termbox:", err)
		return 0, 0
	}
	defer termbox.Close()

	totalScore := 0
	fmt.Println("Random Combo Mode: Solve 10 random combos (each with 6 arrows)!")
	for i := 0; i < count; i++ {
		seq := randomArrows(6)
		completed, _ := processSequence(seq, &totalScore, "Random")
		if !completed {
			fmt.Printf("You exited early. Final Score: %d\n", totalScore)
			return totalScore, time.Since(startTime).Seconds()
		}
	}
	return totalScore, time.Since(startTime).Seconds()
}

// playTimedJSONCombos processes count random JSON combos under an overall time limit.
// The user has the given duration (e.g. 30 seconds) to complete as many combos as possible.
// Each combo earns bonus points if completed quickly.
// Returns total score and elapsed time.
func playTimedJSONCombos(count int, timeLimit time.Duration) (int, float64) {
	overallDeadline := time.Now().Add(timeLimit)
	startTime := time.Now()
	if err := termbox.Init(); err != nil {
		fmt.Println("Failed to initialize termbox:", err)
		return 0, 0
	}
	defer termbox.Close()

	combos, err := loadCombinations("stratagems.json")
	if err != nil {
		fmt.Printf("Error loading combinations: %s\n", err)
		return 0, 0
	}
	rand.Shuffle(len(combos), func(i, j int) {
		combos[i], combos[j] = combos[j], combos[i]
	})
	if count > len(combos) {
		count = len(combos)
	}

	totalScore := 0
	fmt.Println("Timed JSON Combos Mode: You have 30 seconds to solve 10 random combos!")
	for i := 0; i < count; i++ {
		if time.Now().After(overallDeadline) {
			fmt.Println("Time's up!")
			break
		}
		combo := combos[i]
		seq := arrowSequenceFromCombination(combo.Sequence)
		// Use the timed version of processSequence.
		completed, _, _ := processSequenceTimed(seq, &totalScore, combo.Name, overallDeadline)
		if !completed {
			fmt.Printf("You exited early. Final Score: %d\n", totalScore)
			return totalScore, time.Since(startTime).Seconds()
		}
	}
	return totalScore, time.Since(startTime).Seconds()
}

// randomArrows generates a random sequence of n arrows.
func randomArrows(n int) []Arrow {
	keys := []rune{'U', 'D', 'L', 'R'}
	result := make([]Arrow, n)
	for i := range result {
		rk := keys[rand.Intn(len(keys))]
		result[i] = arrowsMap[rk]
	}
	return result
}

// processSequence is the non-timed version.
// It processes a sequence of arrows, updating the total score.
// Returns (completed, scoreEarned).
func processSequence(sequence []Arrow, totalScore *int, title string) (bool, int) {
	score := 0
	printArrows(sequence, *totalScore, title)
	termbox.Flush()
	for _, arrow := range sequence {
		for {
			ev := termbox.PollEvent()
			if ev.Type == termbox.EventKey {
				if ev.Key == arrow.Key {
					fmt.Println("Correct!")
					score += 20
					break // Move to next arrow.
				} else if ev.Key == termbox.KeyEsc || ev.Ch == 'q' || ev.Key == termbox.KeyCtrlC {
					fmt.Println("Exiting...")
					return false, score
				} else {
					fmt.Println("Wrong key, try again!")
					score -= 5
				}
			} else if ev.Type == termbox.EventError {
				panic(ev.Err)
			}
		}
	}
	*totalScore += score
	return true, score
}

// processSequenceTimed is the timed version used in Option 3.
// It uses a ticker to update the display (showing overall time remaining and combo elapsed time)
// and a channel to receive key events.
// Returns (completed, scoreEarned, comboDuration).
func processSequenceTimed(sequence []Arrow, totalScore *int, title string, overallDeadline time.Time) (bool, int, time.Duration) {
	score := 0
	comboStart := time.Now()
	currentIndex := 0

	events := make(chan termbox.Event)
	go func() {
		for {
			events <- termbox.PollEvent()
		}
	}()

	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for currentIndex < len(sequence) {
		remainingOverall := overallDeadline.Sub(time.Now())
		if remainingOverall <= 0 {
			return false, score, time.Since(comboStart)
		}
		select {
		case ev := <-events:
			if ev.Type == termbox.EventKey {
				if ev.Key == sequence[currentIndex].Key {
					fmt.Println("Correct!")
					score += 20
					currentIndex++
				} else if ev.Key == termbox.KeyEsc || ev.Ch == 'q' || ev.Key == termbox.KeyCtrlC {
					fmt.Println("Exiting...")
					return false, score, time.Since(comboStart)
				} else {
					fmt.Println("Wrong key, try again!")
					score -= 5
				}
			} else if ev.Type == termbox.EventError {
				panic(ev.Err)
			}
		case <-ticker.C:
			printArrowsTimed(sequence, *totalScore, title, overallDeadline, comboStart, currentIndex)
			termbox.Flush()
		}
	}

	// Calculate bonus points based on combo completion time.
	comboDuration := time.Since(comboStart)
	bonus := 0
	switch {
	case comboDuration.Seconds() <= 1:
		bonus = 100
	case comboDuration.Seconds() <= 2:
		bonus = 50
	case comboDuration.Seconds() <= 3:
		bonus = 25
	}
	score += bonus
	*totalScore += score
	return true, score, comboDuration
}

// printArrows displays the arrow art (non-timed version) along with title and current score.
func printArrows(sequence []Arrow, currentScore int, title string) {
	clearConsole()
	fmt.Println("Action:", title)
	fmt.Printf("Current Score: %d\n", currentScore)
	lines := make([]string, 5)
	for _, arrow := range sequence {
		parts := strings.Split(arrow.Art, "\n")
		for i := 0; i < 5; i++ {
			lines[i] += parts[i] + "   "
		}
	}
	for _, line := range lines {
		fmt.Println(line)
	}
	fmt.Println()
	fmt.Println()
}

// printArrowsTimed displays the arrow art along with title, current score, overall time remaining,
// and elapsed time for the current combo. The current arrow is highlighted.
func printArrowsTimed(sequence []Arrow, currentScore int, title string, overallDeadline time.Time, comboStart time.Time, currentIndex int) {
	clearConsole()
	remainingOverall := overallDeadline.Sub(time.Now())
	comboElapsed := time.Since(comboStart)
	fmt.Println("Action:", title)
	fmt.Printf("Current Score: %d\n", currentScore)
	fmt.Printf("Overall Time Remaining: %.1f seconds\n", remainingOverall.Seconds())
	fmt.Printf("Combo Time Elapsed: %.2f seconds\n", comboElapsed.Seconds())

	lines := make([]string, 5)
	for i, arrow := range sequence {
		parts := strings.Split(arrow.Art, "\n")
		for j := 0; j < 5; j++ {
			if i == currentIndex {
				lines[j] += ">>" + parts[j] + "<<   "
			} else {
				lines[j] += parts[j] + "   "
			}
		}
	}
	for _, line := range lines {
		fmt.Println(line)
	}
	fmt.Println()
}

// clearConsole uses ANSI escape sequences to clear the screen.
func clearConsole() {
	fmt.Print("\033[H\033[2J")
}

// arrowSequenceFromCombination converts a string like "UDLR" into a slice of Arrow structs.
func arrowSequenceFromCombination(sequence string) []Arrow {
	var result []Arrow
	for _, char := range sequence {
		if arrow, ok := arrowsMap[char]; ok {
			result = append(result, arrow)
		}
	}
	return result
}
