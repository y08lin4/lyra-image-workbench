package settings

func (update Update) HasChanges() bool {
	return update.SiteName != nil ||
		update.NewAPIBaseURL != nil ||
		update.SystemAPIKey != nil ||
		update.SystemBananaAPIKey != nil ||
		update.ClearSystemAPIKey ||
		update.ClearSystemBananaKey ||
		update.PublicBaseURL != nil ||
		update.DebugEnabled != nil ||
		update.TimeoutSec != nil ||
		update.EpayEnabled != nil ||
		update.EpayAPIURL != nil ||
		update.EpayPID != nil ||
		update.EpayKey != nil ||
		update.ClearEpayKey ||
		update.EpayMethods != nil ||
		update.CreditPriceCents != nil ||
		update.MinTopUpCredits != nil ||
		update.ReferralRewardCredits != nil ||
		update.SMTPEnabled != nil ||
		update.SMTPHost != nil ||
		update.SMTPPort != nil ||
		update.SMTPUser != nil ||
		update.SMTPPassword != nil ||
		update.SMTPPass != nil ||
		update.ClearSMTPPassword ||
		update.ClearSMTPPass ||
		update.SMTPFrom != nil ||
		update.SMTPSecure != nil ||
		update.NewUserInitialCredits != nil ||
		update.DailyFreeCredits != nil
}
