package cmd

// command to search for symbols in the database
// used by using the --search flag
func Init() {
	// Add the search flag to the root command
	RootCmd.Flags().BoolVarP(&searchFlag, "search", "s", false, "Search for symbols in the database")
}

// searchFlag is a boolean flag to indicate if the user wants to search for symbols
var searchFlag bool
