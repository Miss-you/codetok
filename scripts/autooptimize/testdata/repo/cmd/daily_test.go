package cmd

func init() {
	testCmd.Flags().String("test-only", "", "defined only in a test")
}
