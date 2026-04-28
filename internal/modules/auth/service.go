package auth

// Service implements the authorization business logic
type Service struct {
	txnRepo     TransactionRepository
	cardRepo    CardRepository
	accountRepo AccountRepository
	hsmAdapter  HSMAdapter
	privacySvc  PrivacyService
	routingSvc  RoutingService
	logger      Logger
	metrics     MetricsCollector
}

// NewService creates a new authorization service
func NewService(
	txnRepo TransactionRepository,
	cardRepo CardRepository,
	accountRepo AccountRepository,
	hsmAdapter HSMAdapter,
	privacySvc PrivacyService,
	routingSvc RoutingService,
	logger Logger,
	metrics MetricsCollector,
) *Service {
	return &Service{
		txnRepo:     txnRepo,
		cardRepo:    cardRepo,
		accountRepo: accountRepo,
		hsmAdapter:  hsmAdapter,
		privacySvc:  privacySvc,
		routingSvc:  routingSvc,
		logger:      logger,
		metrics:     metrics,
	}
}
