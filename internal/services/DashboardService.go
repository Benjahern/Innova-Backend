package services

import (
	"fmt"
	"time"

	"turno-papa/internal/repository"
)

type DashboardService struct {
	attendanceRepo   repository.AttendanceLogRepository
	weeklyHoursRepo  repository.WeeklyHoursRepository
	monthlyArrearsRepo repository.MonthlyArrearsRepository
	userRepo         repository.UserRepository
}

func NewDashboardService(
	attendanceRepo repository.AttendanceLogRepository,
	weeklyHoursRepo repository.WeeklyHoursRepository,
	monthlyArrearsRepo repository.MonthlyArrearsRepository,
	userRepo repository.UserRepository,
) *DashboardService {
	return &DashboardService{
		attendanceRepo:    attendanceRepo,
		weeklyHoursRepo:   weeklyHoursRepo,
		monthlyArrearsRepo: monthlyArrearsRepo,
		userRepo:          userRepo,
	}
}

type DashboardSummary struct {
	TotalWorkers    int             `json:"total_workers"`
	WorkersPresent int             `json:"workers_present"`
	TotalHoursWeek float64         `json:"total_hours_week"`
	TotalArrearsMin int             `json:"total_arrears_minutes"`
	ByWorker        []WorkerSummary `json:"by_worker"`
}

type WorkerSummary struct {
	UserID      string    `json:"user_id"`
	Name        string    `json:"name"`
	HoursWeek   float64   `json:"hours_week"`
	ArrearsMin  int       `json:"arrears_minutes"`
	Checkins    int       `json:"checkins"`
	LastCheckin *time.Time `json:"last_checkin,omitempty"`
}

// GetDashboard returns dashboard summary for a company
// Summaries are pre-calculated by DB triggers, this just aggregates
func (s *DashboardService) GetDashboard(companyID string) (*DashboardSummary, error) {
	workers, err := s.userRepo.GetByCompany(companyID)
	if err != nil {
		return nil, fmt.Errorf("get workers: %w", err)
	}

	now := time.Now()
	today := now.Format("2006-01-02")

	// Get week start (Monday)
	weekday := int(now.Weekday())
	if weekday == 0 {
		weekday = 7
	}
	weekStart := now.AddDate(0, 0, -weekday+1)

	summary := &DashboardSummary{
		TotalWorkers: len(workers),
		ByWorker:     make([]WorkerSummary, 0, len(workers)),
	}

	for _, worker := range workers {
		ws := WorkerSummary{
			UserID: worker.UserID,
			Name:   worker.Name,
		}

		// Get today's attendance to count present workers
		if worker.BranchID != nil {
			todayLogs, err := s.attendanceRepo.GetByBranchAndDate(*worker.BranchID, today)
			if err == nil {
				for _, log := range todayLogs {
					if log.UserID == worker.UserID {
						ws.Checkins++
						if ws.LastCheckin == nil || log.Timestamp.After(*ws.LastCheckin) {
							ws.LastCheckin = &log.Timestamp
						}
					}
				}
			}
		}

		// Get pre-calculated weekly hours (DB auto-updates via trigger)
		weekHours, err := s.weeklyHoursRepo.GetByUserAndWeek(worker.UserID, weekStart)
		if err == nil && weekHours != nil {
			ws.HoursWeek = weekHours.TotalHours
			summary.TotalHoursWeek += weekHours.TotalHours
		}

		// Get pre-calculated monthly arrears (DB auto-updates via trigger)
		arrears, err := s.monthlyArrearsRepo.GetByUserAndMonth(worker.UserID, now.Year(), int(now.Month()))
		if err == nil && arrears != nil {
			ws.ArrearsMin = arrears.TotalArrearsMin
			summary.TotalArrearsMin += arrears.TotalArrearsMin
		}

		summary.ByWorker = append(summary.ByWorker, ws)
	}

	summary.WorkersPresent = countWorkersPresent(summary.ByWorker)

	return summary, nil
}

func countWorkersPresent(workers []WorkerSummary) int {
	count := 0
	for _, w := range workers {
		if w.Checkins > 0 {
			count++
		}
	}
	return count
}