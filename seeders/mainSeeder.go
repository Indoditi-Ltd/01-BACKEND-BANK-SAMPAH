package seeders

func SeedAll() error {
	// Seed roles terlebih dahulu
	if err := SeedRole(); err != nil {
		return err
	}
	if err := SeedParentBank(); err != nil {
		return err
	}
	if err := SeedPlan(); err != nil {
		return err
	}
	if err := SeedDivision(); err != nil {
		return err
	}
	if err := SeedUsers(); err != nil {
		return err
	}
	return nil
}
