---
trigger: always_on
---

1. Cấu trúc Thư mục chuẩn (Standard Go Layout)

Đừng để AI ném tất cả vào thư mục main. Hãy ép nó tuân thủ cấu trúc thư mục chuẩn của cộng đồng Go:

.
├── cmd/                         # Điểm entry của ứng dụng
│   └── server/
│       └── main.go              # Nơi khởi tạo và chạy App
├── internal/                    # Code riêng tư, không cho phép package ngoài import
│   ├── app/                     # Wire up: Kết nối database, repository, service
│   ├── domain/                  # Interface và Business Entities (Quan trọng nhất)
│   ├── repository/              # Truy vấn Database (GORM, SQLX, v.v.)
│   ├── service/                 # Logic nghiệp vụ (Trái tim của hệ thống)
│   └── handler/                 # Tầng giao tiếp (HTTP/gRPC Handlers)
├── pkg/                         # Code có thể dùng chung cho các dự án khác
├── api/                         # File định nghĩa API (Swagger/OpenAPI, Proto)
├── configs/                     # Cấu hình hệ thống (YAML, ENV)
├── go.mod
└── go.sum

# Go Development Rules
- **Error Handling**: Never ignore errors. Check `if err != nil` immediately. No `panic()` in production code.
- **Interfaces**: Define interfaces where they are USED (consumer side), not where they are implemented.
- **Concurrency**: Use channels and goroutines only when necessary. Ensure no goroutine leaks.
- **Naming**: 
    - Use `camelCase` for private, `PascalCase` for public.
    - Receiver names should be short (1-3 letters), e.g., `func (r *Repository)`.
- **Context**: Always pass `context.Context` as the first argument in functions involving I/O or DB.
- **Dependencies**: Use Dependency Injection. Do not use `init()` functions or global variables for DB connections.

Cách AI phối hợp giữa Flutter & Go

Đây là điểm "ăn tiền" khi dùng Orchestrator. Khi bạn thêm một tính năng mới, hãy dùng Prompt này:

    "Tôi muốn thêm tính năng 'Quên mật khẩu'.

        Đầu tiên, hãy thiết kế API Endpoint trong Golang (Handler -> Service -> Repository).

        Sau đó, dựa trên JSON response của Go, hãy cập nhật Flutter (Data Source -> Repository -> BLoC).

        Hãy đảm bảo kiểu dữ liệu DateTime giữa hai bên đồng nhất (RFC3339)."

4. Tips nâng cao cho Golang

    Pointer vs Value Receiver: AI thường bối rối khi nào dùng (s *Service) và (s Service). Hãy nhắc nó: "Dùng pointer receiver cho các struct có trạng thái hoặc khi cần thay đổi dữ liệu bên trong."

    Tags cho Struct: Go rất mạnh về Metadata. Nhắc AI: "Luôn thêm json:"key_name" và db:"column_name" cho các struct để tránh lỗi mapping."

    Unit Test: Go có bộ công cụ test đi kèm rất mạnh. Hãy ra lệnh: "Mỗi khi viết một hàm Service mới, hãy viết kèm một file _test.go sử dụng testify và mockery."

Về Error Handling: * Nên thêm: "Sử dụng %w trong fmt.Errorf để wrap lỗi, giúp giữ được stack trace hoặc ngữ cảnh của lỗi cũ."

Về Domain-Driven Design (DDD) trong internal/domain:

    Nên thêm: "Thư mục domain không được phép import bất kỳ package nào từ repository, service hay handler (để tránh vòng lặp - cyclic dependency)." Đây là lỗi AI rất hay mắc khi thiết kế hệ thống.

Về Performance (Slice/Map):

    Nên thêm: "Ưu tiên cấp phát trước bộ nhớ (pre-allocate) bằng make([]Type, 0, capacity) nếu biết trước kích thước xấp xỉ của slice để tối ưu performance."

Về Logging:

    Nên thêm: "Không sử dụng fmt.Println. Hãy sử dụng structured logging (như slog của Go 1.21+ hoặc zap) để ghi log dưới dạng JSON."