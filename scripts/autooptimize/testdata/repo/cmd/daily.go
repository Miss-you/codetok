package cmd

func init() {
	dailyCmd.Flags().Bool("json", false, "JSON output")
	dailyCmd.Flags().Int("days", 7, "rolling window")
	dailyCmd.Flags().String("since", "", "start date")
	loginCmd.Flags().StringVar(&token, "token", "", "session token")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "verbose")
	_, _ = dailyCmd.Flags().GetString("lookup-only")
}
