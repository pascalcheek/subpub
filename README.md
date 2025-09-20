# PubSub Service

Микросервис для подписки и публикации событий по gRPC

##  Структура проекта
```
subpub/
├── cmd/
│   └── server/
│       └── main.go          # Точка входа
├── configs/
│   └── config.yaml          # Конфигурация
├── internal/
│   ├── config/
│   │   └── config.go        # Загрузка конфигурации
│   ├── grpc/
│   │   ├── proto/           # Protobuf схемы
│   │   │   ├── gen/         # Сгенерированный код
│   │   │   └── pubsub.proto # API определение
│   │   └── service.go       # Реализация gRPC сервиса
│   └── subpub/
│       └── subpub.go        # Ядро pub-sub системы
├── go.mod
└── go.sum
```
## Детали реализации

### 1. Ядро системы (internal/subpub/subpub.go)
```
// SubPub - основной интерфейс системы
type SubPub interface {
    Subscribe(subject string, cb MessageHandler) (Subscription, error)
    Publish(subject string, msg interface{}) error
    Close(ctx context.Context) error
}

// Реализация:
type subPubImpl struct {
    mu          sync.RWMutex
    subjects    map[string][]*subscription
    publishChan chan publishRequest
    // ...
}

// Особенности:
// - Потокобезопасность через sync.RWMutex
// - Буферизованный канал для публикации (размер 100)
// - Отдельная горутина для обработки сообщений
// - Graceful shutdown с таймаутом
```
### 2. gRPC сервис (internal/grpc/service.go)
```
type PubSubService struct {
    pb.UnimplementedPubSubServer
    bus     subpub.SubPub
    streams map[string][]pb.PubSub_SubscribeServer
    mu      sync.Mutex
}

// Subscribe:
// 1. Создает подписку в subpub
// 2. Запускает горутину для отправки сообщений клиенту
// 3. Следит за контекстом клиента

// Publish:
// 1. Валидирует запрос
// 2. Передает сообщение в subpub
// 3. Возвращает Empty

// Особенности:
// - Поддержка множества подписчиков на один ключ
// - Автоматическая отписка при отключении клиента
// - Потокобезопасность через mutex
```

### 3. Конфигурация (internal/config/config.go)
```
type Config struct {
    GRPC struct {
        Port int `yaml:"port" env:"GRPC_PORT" env-default:"50051"`
    } `yaml:"grpc"`
    Log struct {
        Level string `yaml:"level" env:"LOG_LEVEL" env-default:"info"`
    } `yaml:"log"`
}

// Load загружает конфиг:
// 1. Ищет config.yaml в ./configs/
// 2. Проверяет переменные окружения
// 3. Применяет значения по умолчанию
```
### 4. Точка входа (cmd/server/main.go)
```
func main() {
    // 1. Загрузка конфигурации
    cfg := config.Load() 
    
    // 2. Настройка логгера
    logger := setupLogger(cfg.Log.Level)
    
    // 3. Инициализация SubPub
    bus := subpub.New()
    
    // 4. Создание gRPC сервера
    server := grpc.NewServer(
        grpc.KeepaliveParams(keepalive.ServerParameters{
            MaxConnectionIdle: 5 * time.Minute,
        }),
    )
    
    // 5. Регистрация сервиса
    pb.RegisterPubSubServer(server, service.New(bus, logger))
    
    // 6. Graceful shutdown
    go handleShutdown(server, bus)
    
    // 7. Запуск сервера
    lis, _ := net.Listen("tcp", fmt.Sprintf(":%d", cfg.GRPC.Port))
    server.Serve(lis)
}
```
## Best Practices
Graceful Shutdown, Dependency Injection

##  Быстрый старт

### Требования
- Go 1.21+
- Protobuf компилятор

```bash
# 1. Клонировать репозиторий
git clone https://github.com/pascalcheek/subpub.git
cd subpub

# 2. Установить зависимости
go mod download

# 3. Запустить сервер
go run cmd/server/main.go
