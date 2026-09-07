package services

type PlanLimits struct {
	RequestsPerDay    int
	CharactersPerDay  int
	RequestsPerMinute int
}

func GetPlanLimits(plan string) PlanLimits {
	switch plan {
	case "admin":
		return PlanLimits{
			RequestsPerDay:    0,
			CharactersPerDay:  0,
			RequestsPerMinute: 0,
		}

	case "free_24h":
		return PlanLimits{
			RequestsPerDay:    20,
			CharactersPerDay:  10000,
			RequestsPerMinute: 5,
		}

	case "free_30d":
		return PlanLimits{
			RequestsPerDay:    50,
			CharactersPerDay:  25000,
			RequestsPerMinute: 5,
		}

	case "linkedin":
		return PlanLimits{
			RequestsPerDay:    20,
			CharactersPerDay:  10000,
			RequestsPerMinute: 5,
		}

	default:
		return PlanLimits{
			RequestsPerDay:    10,
			CharactersPerDay:  5000,
			RequestsPerMinute: 3,
		}
	}
}
