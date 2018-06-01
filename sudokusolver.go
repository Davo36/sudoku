// A Sudoku solver

// DNB31May2018: This seems to be working now.  Things to add to it:

// 1) Read in some arguments. Map number etc.
// 2) We store the whole board in each state, should just need to store the move location and it's value.
// 3) GUI's used by go?

package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
	"tools"
)

// Just a bunch of general constants
const (
	maxBranchFactor = 9         // Max no. of child nodes from any parent node.
	maxIterations   = 600000000 // Max no. of iterations around main loop.
	xDim            = 9         // Max X dimension of puzzle.
	yDim            = 9         // Max Y dimension of puzzle.

	debug = false
)

// The solve can finish in various ways
type endState int

const (
	eNotFinished                      endState = 0
	eSolutionFound                    endState = 1 // A solution was found and the prog stopped.
	eAllMovesSearchedSolutionFound    endState = 2 // Exhaustive search done, all solutions returned.
	eAllMovesSearchedNoSolutionFound  endState = 3 // Damn!
	eMaxNoOfIterationsSolutionFound   endState = 4 // Run out of iterations but at least 1 soln found.
	eMaxNoOfIterationsNoSolutionFound endState = 5 // Run out of iterations, no soln found.
	eSolveCancelled                   endState = 6 // True if user cancels solve.
)

// Coord is a simple 2d coordy type
type Coord struct {
	x, y int
}

// GameState is a state struct that represents a board position.
type GameState struct {
	boardValues [xDim][yDim]int // The values on the board.

	xCoord int // The x coordinate of the position we're making the move in.
	yCoord int // ditto for y

	newValue int // The new value being entered in.

	depth int // The depth of this node in the tree.

	score int // How good is this position?  Need a signed quantity.

}

// Puzzle is the main struct in this program.
type Puzzle struct {
	maxIterations int // Num times to go around main event loop.
	solutionFound bool
	numSolutions  int
	showMoves     bool // True if we are going to update the screen as we solve.

	cancelSolve    bool // True if the solve has been cancelled.
	puzzleEndState endState
	logFile        bool
	clog           *os.File // A debug file. clog = console log.
	mapFileName    string   // The puzzle name we are currently solving.

	totalMoves           int
	world                [xDim][yDim]int // A 2D array representation of the current state of our world, each square is FREE_SQUARE, BOX_SQUARE, WALL_SQUARE.
	gameStatesTriedCount int             // The number of game states that have been tried so far.
	gameStatesFoundCount int             // The number of game states that have been generated so far.
	numStatesRemoved     int             // States that led to nothing and were removed.

	availables             [xDim * yDim][maxBranchFactor]GameState // Depth first search data structures.
	numberOfAvailables     [xDim * yDim]int
	numberOfAvailablesCopy [xDim * yDim]int
	numBlankSquares        int
	blankSquareCoords      [xDim * yDim]Coord
	blanksSquares2D        [xDim][yDim]bool
}

//---------------------------------------------------------------------------------------------
// This function just outputs a text representation of the world we have created to the
// debug file so we can see what the world array etc. looks like.
//---------------------------------------------------------------------------------------------
func (puz *Puzzle) outputWorld(toLogFile bool, toScreen bool) {

	if toLogFile {
		puz.clog.WriteString("\nBoard:\n")
		for i := 0; i < yDim; i++ {
			outputStr := fmt.Sprintf("%v", puz.world[i])
			puz.clog.WriteString(outputStr + "\n")
		}
	}

	if toScreen {
		fmt.Printf("\nBoard:\n")
		for i := 0; i < yDim; i++ {
			outputStr := fmt.Sprintf("%v", puz.world[i])
			fmt.Printf(outputStr + "\n")
		}
	}

} // outputWorld

// Initialise our variables used in the solve
func (puz *Puzzle) initVars() {

	puz.maxIterations = 0
	puz.solutionFound = false
	puz.numSolutions = 0

	puz.cancelSolve = false
	puz.puzzleEndState = eNotFinished
	puz.logFile = false
	puz.mapFileName = "0"

	for i := 0; i < xDim; i++ {
		for j := 0; j < yDim; j++ {
			puz.world[i][j] = 0
			puz.blanksSquares2D[i][j] = false
		}
	}

	for i := 0; i < xDim*yDim; i++ {
		puz.numberOfAvailables[i] = 0
		puz.numberOfAvailablesCopy[i] = 0
	}
	puz.numBlankSquares = 0

	puz.totalMoves = 1

	puz.gameStatesTriedCount = 0
	puz.gameStatesFoundCount = 0
	puz.numStatesRemoved = 0

} // InitVars

//---------------------------------------------------------------------------------------------
// This is the crux of it!  All legal moves are generated and stored in the availables array.
//---------------------------------------------------------------------------------------------
func (puz *Puzzle) getAllLegalMoves(theGameState GameState) int {

	numMovesFound := 0

	if debug && puz.logFile {
		puz.clog.WriteString("sub getAllLegalMoves\n")
		outputStr := fmt.Sprintf("Depth = %d\n", theGameState.depth)
		puz.clog.WriteString(outputStr)
	}

	for i := 0; i < xDim; i++ {
		for j := 0; j < yDim; j++ {

			if puz.world[i][j] == 0 {
				for k := 1; k < xDim+1; k++ {

					if puz.boardIsLegal(k, i, j) {
						// Have a new move.
						puz.availables[puz.numBlankSquares][puz.numberOfAvailables[puz.numBlankSquares]] = theGameState // This is a copy

						puz.availables[puz.numBlankSquares][puz.numberOfAvailables[puz.numBlankSquares]].boardValues[i][j] = k
						puz.availables[puz.numBlankSquares][puz.numberOfAvailables[puz.numBlankSquares]].depth = theGameState.depth + 1
						puz.availables[puz.numBlankSquares][puz.numberOfAvailables[puz.numBlankSquares]].xCoord = i
						puz.availables[puz.numBlankSquares][puz.numberOfAvailables[puz.numBlankSquares]].yCoord = j
						puz.availables[puz.numBlankSquares][puz.numberOfAvailables[puz.numBlankSquares]].newValue = k

						puz.numberOfAvailables[puz.numBlankSquares]++
						puz.numberOfAvailablesCopy[puz.numBlankSquares]++
						numMovesFound++

						puz.blankSquareCoords[puz.numBlankSquares].x = i
						puz.blankSquareCoords[puz.numBlankSquares].y = j

						puz.blanksSquares2D[i][j] = true

					}
				}
				puz.numBlankSquares++
			}

		}
	}

	if numMovesFound == 0 {
		if debug && puz.logFile {
			puz.clog.WriteString("No moves available\n\n")
		}
		return puz.numBlankSquares
	}

	// Print out the available moves generated.
	if debug && puz.logFile {
		puz.clog.WriteString("Availables are: \n")
		iIndex := 0
		for i := 0; i < puz.numBlankSquares; i++ {
			for j := 0; j < puz.numberOfAvailables[i]; j++ {
				x := puz.availables[i][j].xCoord
				y := puz.availables[i][j].yCoord
				iIndex++
				outputStr := fmt.Sprintf("%2d)  %d at (%d, %d)\n", iIndex, puz.availables[i][j].boardValues[x][y], x, y)
				puz.clog.WriteString(outputStr)
			}
		}
		outputStr := fmt.Sprintf("\nNum blank squares = %d\n\n", puz.numBlankSquares)
		puz.clog.WriteString(outputStr)
	}

	return puz.numBlankSquares

} // getAllLegalMoves

//---------------------------------------------------------------------------------------------
// Output a heading in the output window
//---------------------------------------------------------------------------------------------
func (puz *Puzzle) mainDebugHeading() {
	puz.clog.WriteString("\nNum      Steps    States   States     Diff")
	puz.clog.WriteString("\nIts      Soln     Found    Tried")
	puz.clog.WriteString("\n------------------------------------------\n")
} // MainDebugHeading

//---------------------------------------------------------------------------------------------
// Output some debug each time around main loop.
//---------------------------------------------------------------------------------------------
func (puz *Puzzle) mainDebug(theGameState *GameState, sw *tools.StopWatch, listLength int) {

	outputStr := fmt.Sprintf("%8d  %4d  %8d %8d %8d", puz.totalMoves, theGameState.depth, puz.gameStatesFoundCount,
		puz.gameStatesTriedCount, listLength)
	puz.clog.WriteString(outputStr)

	if puz.totalMoves%100000 == 0 {
		// Renew headings
		tempTime := sw.EndTimer()

		searchSpeed := 1000000
		if tempTime > 0 {
			searchSpeed = int(float64(puz.gameStatesTriedCount) / tempTime.Seconds())
		}

		outputStr = fmt.Sprintf("Speed = %d states p/s.", searchSpeed)
		puz.clog.WriteString(outputStr)
		outputStr = fmt.Sprintf("Elapsed Time: %4.2f secs.\n", tempTime.Seconds())
		puz.clog.WriteString(outputStr)
		puz.mainDebugHeading()

		if puz.totalMoves%1000000 == 0 {
			// Write out state of world
			puz.outputWorld(true, false)
		}

	}

} // mainDebug

//---------------------------------------------------------------------------------------------
// Check if a board position is legal.
// Returns false if board breaks a rule, otherwise true.
//---------------------------------------------------------------------------------------------
func (puz *Puzzle) boardIsLegal(k int, i int, j int) bool {

	// Check row i
	for x := 0; x < xDim; x++ {
		if puz.world[x][j] == k && x != i {
			return false
		}
	}

	// Check column j
	for y := 0; y < yDim; y++ {
		if puz.world[i][y] == k && !(y == j) {
			return false
		}
	}

	// Check the square it's in.
	// 0 1 2
	// 1
	// 2
	iXCoord := i / 3
	iYCoord := j / 3

	iStartX := iXCoord * 3
	iStartY := iYCoord * 3

	for x := iStartX; x < iStartX+3; x++ {
		for y := iStartY; y < iStartY+3; y++ {
			if puz.world[x][y] == k && x != i && y != j {
				return false
			}
		}
	}

	return true

} // BoardIsLegal

//---------------------------------------------------------------------------------------------
// This routine tests whether we have found the solution or not
//---------------------------------------------------------------------------------------------
func (puz *Puzzle) testSolution() bool {
	//TODO: Don't need to check every square.
	for i := 0; i < xDim; i++ {
		for j := 0; j < yDim; j++ {
			if puz.world[i][j] == 0 {
				return false
			}
			if !puz.boardIsLegal(puz.world[i][j], i, j) {
				return false
			}
		}
	}

	if puz.logFile {
		puz.clog.WriteString("Solution found.\n")
	}

	return true

} // TestSolution

//---------------------------------------------------------------------------------------------
// Put the next available move into our data structures.
//---------------------------------------------------------------------------------------------
func (puz *Puzzle) addNextAvailable(blankSquareNum int) (bool, int) {

	// Move on to the next blank square.
	if blankSquareNum >= puz.numBlankSquares {
		return false, blankSquareNum
	}

	if debug && puz.logFile {
		outputStr := fmt.Sprintf("\nEntering sub AddNextAvailable. Square no. %d, available no. %d.\n",
			blankSquareNum, puz.numberOfAvailablesCopy[blankSquareNum])
		puz.clog.WriteString(outputStr)
	}

	// Update the  world data structures
	if puz.numberOfAvailablesCopy[blankSquareNum] > 0 {
		iNextAvailable := puz.numberOfAvailablesCopy[blankSquareNum] - 1
		iXCoord := puz.availables[blankSquareNum][iNextAvailable].xCoord
		iYCoord := puz.availables[blankSquareNum][iNextAvailable].yCoord
		if debug && puz.logFile {
			outputStr := fmt.Sprintf("Trying to add: Value %d in (%d,%d).\n",
				puz.availables[blankSquareNum][iNextAvailable].newValue, iXCoord, iYCoord)
			puz.clog.WriteString(outputStr)
		}
		if puz.boardIsLegal(puz.availables[blankSquareNum][iNextAvailable].newValue, iXCoord, iYCoord) {
			puz.world[iXCoord][iYCoord] = puz.availables[blankSquareNum][iNextAvailable].newValue
			puz.gameStatesTriedCount++
			if debug && puz.logFile {
				outputStr := fmt.Sprintf("Added move: Value %d in (%d,%d).\n",
					puz.availables[blankSquareNum][iNextAvailable].newValue, iXCoord, iYCoord)
				puz.clog.WriteString(outputStr)
			}
			// Throw this available away as we have used it.
			puz.numberOfAvailablesCopy[blankSquareNum]--
			// Move on to the next blank square.
			blankSquareNum++
			return true, blankSquareNum
		}

		if debug && puz.logFile {
			puz.clog.WriteString("Available not added as it is an illegal move.\n")
		}
		return false, blankSquareNum

	}

	if debug && puz.logFile {
		puz.clog.WriteString("No moves to add.\n")
	}
	return false, blankSquareNum

} // AddNextAvailable

//---------------------------------------------------------------------------------------------
// Remove the last available and update data structures.
//---------------------------------------------------------------------------------------------
func (puz *Puzzle) removeAvailable(blankSquareNum int) int {

	iXCoord := puz.blankSquareCoords[blankSquareNum].x
	iYCoord := puz.blankSquareCoords[blankSquareNum].y

	if debug && puz.logFile {
		outputStr := fmt.Sprintf("Entering sub RemoveAvailable. Square no. %d at (%d,%d).\n", blankSquareNum, iXCoord, iYCoord)
		puz.clog.WriteString(outputStr)
	}

	// Clear the world array
	puz.world[iXCoord][iYCoord] = 0

	if debug && puz.logFile {
		outputStr := fmt.Sprintf("Removed move: Cleared (%d,%d).\n", iXCoord, iYCoord)
		puz.clog.WriteString(outputStr)
	}

	puz.numberOfAvailablesCopy[blankSquareNum]--
	if puz.numberOfAvailablesCopy[blankSquareNum] <= 0 {
		if debug && puz.logFile {
			outputStr := fmt.Sprintf("Out of availables for square %d. Moving back to square %d.\n", blankSquareNum, blankSquareNum-1)
			puz.clog.WriteString(outputStr)
		}
		puz.numberOfAvailablesCopy[blankSquareNum] = puz.numberOfAvailables[blankSquareNum]
		blankSquareNum--
	}

	return blankSquareNum

} // removeAvailable

//---------------------------------------------------------------------------------------------
// Do the actual search
//---------------------------------------------------------------------------------------------
func (puz *Puzzle) depthFirstSearch(startGameState *GameState) bool {

	allMovesSearched := false
	var sw tools.StopWatch

	puz.numBlankSquares = puz.getAllLegalMoves(*startGameState)

	iBlankSquareNum := 0
	bFinished := false

	for !bFinished && puz.totalMoves < puz.maxIterations && !puz.cancelSolve && !puz.solutionFound {

		ret := false
		ret, iBlankSquareNum = puz.addNextAvailable(iBlankSquareNum)
		if ret {
			iNextAvailable := puz.numberOfAvailablesCopy[iBlankSquareNum] - 1

			if debug && puz.logFile {
				puz.outputWorld(true, false)
			}
			if puz.totalMoves%10000 == 0 {
				puz.mainDebug(&puz.availables[iBlankSquareNum][iNextAvailable], &sw, 100) //  No. of remaining states to try?
			}

			// DNB29May18: Can we just do something like: if iBlankSquareNum == numBlankSquares...
			if puz.testSolution() {
				puz.solutionFound = true
				puz.numSolutions++
				allMovesSearched = false
				return allMovesSearched
			}

		} else {

			// Remove this available and shift onto the next one.
			iBlankSquareNum = puz.removeAvailable(iBlankSquareNum)
			if debug && puz.logFile {
				puz.outputWorld(true, false)
			}
			if iBlankSquareNum < 0 {
				bFinished = true
			}
		}

		// Increment the loop counter
		puz.totalMoves++

	}

	allMovesSearched = true

	return allMovesSearched

} // depthFirstSearch

//---------------------------------------------------------------------------------------------
// This routine just copies the world data into our first game state.
//---------------------------------------------------------------------------------------------
func (puz *Puzzle) storeStartPosition(startGameState *GameState) {

	// Copy the world into our initial game state.
	startGameState.boardValues = puz.world // TODO: Is this a copy or reference.  We want a copy.
	startGameState.depth = 1
	puz.gameStatesFoundCount = 1

}

//---------------------------------------------------------------------------------------------
// solve
//---------------------------------------------------------------------------------------------
func (puz *Puzzle) solve() (bool, time.Duration) {

	// Set up the problem data structures
	var sw tools.StopWatch
	sw.StartTimer()
	var startGameState GameState

	// Store the starting position
	puz.storeStartPosition(&startGameState)
	puz.outputWorld(true, true)

	puz.mainDebugHeading()
	allMovesSearched := puz.depthFirstSearch(&startGameState)

	// How did we end? Save the status for debug purposes.
	puz.puzzleEndState = eSolutionFound
	if puz.cancelSolve {
		puz.puzzleEndState = eSolveCancelled
	} else {
		if allMovesSearched {
			puz.puzzleEndState = eAllMovesSearchedSolutionFound
			if !puz.solutionFound {
				puz.puzzleEndState = eAllMovesSearchedNoSolutionFound
			}
		} else {
			if puz.totalMoves == puz.maxIterations {
				puz.puzzleEndState = eMaxNoOfIterationsSolutionFound
				if !puz.solutionFound {
					puz.puzzleEndState = eMaxNoOfIterationsNoSolutionFound
				}
			}
		}
	}

	timeTaken := sw.EndTimer()

	// Some very final debug
	if puz.solutionFound {
		outputStr := fmt.Sprintf("\nSolution (%d) found: \n", puz.numSolutions)
		puz.clog.WriteString(outputStr)
		fmt.Printf(outputStr)
		puz.outputWorld(true, true)
		return true, timeTaken
	}

	return false, timeTaken

} // solve

//---------------------------------------------------------------------------------------------
// Do some initialisation stuff
//---------------------------------------------------------------------------------------------
func (puz *Puzzle) setup(numIterations int, mapFileName string, logFile bool) {

	puz.initVars()

	// Open a log file.
	openedFile, err := os.Create("logFile.txt")
	if err == nil {
		puz.clog = openedFile
	} else {
		puz.logFile = false
	}

	puz.maxIterations = tools.Min(tools.Max(numIterations, 0), maxIterations)
	puz.logFile = logFile

	puz.clog.WriteString("-----------------------------------------------------------------\n")
	fmt.Println("-----------------------------------------------------------------")
	outputStr := fmt.Sprintf("SuDoKu solve as at: %v\n", time.Now())
	puz.clog.WriteString(outputStr)
	fmt.Printf(outputStr)
	puz.mapFileName = mapFileName
	outputStr = fmt.Sprintf("Puzzle Number: %s\n", puz.mapFileName)
	puz.clog.WriteString(outputStr)
	fmt.Printf(outputStr)
	puz.clog.WriteString("Algorithm: Depth First Search.\n")
	outputStr = fmt.Sprintf("Max iterations: %d\n", puz.maxIterations)
	puz.clog.WriteString(outputStr)

	// Load the map into our 2d world array
	absPath, _ := filepath.Abs("../sudoku/maps/" + mapFileName)
	data, _ := ioutil.ReadFile(absPath)
	strData := strings.Replace(string(data), "\n", "", -1)
	strData = strings.Replace(strData, "\r", " ", -1)
	strData = strings.Replace(strData, "  ", " ", -1)
	individualStrings := strings.Split(strData, " ")
	for i := 0; i < xDim; i++ {
		for j := 0; j < yDim; j++ {
			entry, _ := strconv.Atoi(individualStrings[i*xDim+j])
			puz.world[i][j] = entry
		}
	}

	puz.clog.WriteString("This is the starting position:\n")
	fmt.Println("This is the starting position:")
	for i := 0; i < yDim; i++ {
		outputStr = fmt.Sprintf("%v", puz.world[i])
		puz.clog.WriteString(outputStr + "\n")
		fmt.Println(outputStr)
	}

}

//---------------------------------------------------------------------------------------------
// Run baby run.
//---------------------------------------------------------------------------------------------
func main() {

	// Need to get some arguments from the command line
	// These are temp ones

	numIterations := 100000
	mapFileName := "53.txt"
	logFile := false

	var puz Puzzle
	puz.setup(numIterations, mapFileName, logFile)

	solved, timeTaken := puz.solve()

	if solved {
		outputStr := fmt.Sprintf("\nSolved in %4.2f seconds.\n", timeTaken.Seconds())
		puz.clog.WriteString(outputStr + "\n")
		fmt.Println(outputStr)
	} else {
		puz.clog.WriteString("No solution found.\n")
		fmt.Printf("No solution found.\n")
	}

	puz.clog.Close()

}
