# Swagger API Documentation

## Tổng quan

Dự án đã được tích hợp Swagger để quản lý và test API một cách dễ dàng.

## Truy cập Swagger UI

Sau khi chạy backend server, truy cập Swagger UI tại:
```
http://localhost:8080/swagger/index.html
```

## Các API Endpoints

### Authentication (Public)
- `POST /auth/register` - Đăng ký user mới
- `POST /auth/login` - Đăng nhập và nhận JWT token

### User Management (Protected - cần JWT token)
- `GET /api/users/me` - Lấy thông tin user hiện tại
- `PUT /api/users/me` - Cập nhật thông tin user
- `PUT /api/users/me/password` - Đổi mật khẩu

## Sử dụng JWT Token trong Swagger

1. Đăng nhập qua endpoint `/auth/login` để nhận token
2. Click vào nút **Authorize** ở góc trên bên phải Swagger UI
3. Nhập: `Bearer <your-token>` (ví dụ: `Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...`)
4. Click **Authorize** và **Close**
5. Bây giờ bạn có thể test các protected endpoints

## Generate Swagger Docs

Để generate lại Swagger documentation sau khi thay đổi code:

```bash
# Cách 1: Sử dụng swag CLI (nếu đã cài đặt)
swag init -g cmd/api/main.go -o docs

# Cách 2: Sử dụng go run
go run github.com/swaggo/swag/cmd/swag@latest init -g cmd/api/main.go -o docs

# Cách 3: Sử dụng script
./run.sh
```

## Chạy Backend

```bash
# Cách 1: Sử dụng script
./run.sh

# Cách 2: Chạy trực tiếp
go run cmd/api/main.go
```

## Cấu hình

Đảm bảo file `configs/config.yaml` được cấu hình đúng:
- Database connection string
- JWT secret key
- Server port (mặc định: :8080)

## Lưu ý

- Swagger docs được tự động generate khi chạy `./run.sh`
- Nếu thay đổi Swagger annotations, cần generate lại docs
- Tất cả protected endpoints yêu cầu JWT token trong header `Authorization`

# Setup PostgreSQL

## Cài PostgreSQL (Arch / EndeavourOS)
sudo pacman -S postgresql
- Kiểm tra : pacman -Qs postgresql

## Khởi tạo database lần đầu :
sudo -iu postgres
initdb --locale=C.UTF-8 --encoding=UTF8 -D /var/lib/postgres/data
exit

## Start & enable PostgreSQL service
sudo systemctl start postgresql
sudo systemctl enable postgresql
- Kiểm tra : systemctl status postgresql

## Tạo database cho dự án camera_security
sudo -iu postgres
psql
CREATE DATABASE camera_security;
\q
exit

## (Tuỳ chọn) Set password cho user postgres
sudo -iu postgres
psql
ALTER USER postgres WITH PASSWORD '123456';

## Test kết nối thủ công
psql -h localhost -U postgres -d camera_security


Mục tiêu : Chuẩn bị PostgreSQL để Golang API có thể kết nối và chạy được.
Nhớ rằng đây là chạy DB local, muốn không phụ thuộc có Dockerfile

# Cài Docker trên Arch / EndeavourOS (đúng chuẩn)

sudo pacman -S docker docker-compose
sudo systemctl enable --now docker
systemctl status docker

## (QUAN TRỌNG) Cho user chạy docker không cần sudo

### Mặc định Docker cần root
sudo usermod -aG docker $USER

### Logout / reboot hoặc:
newgrp docker

## Test
docker version
docker compose version

## Chạy lại project
docker compose up --build
docker compose up --build -d

## Nếu lỗi và sửa nhiều 
docker compose down -v
docker compose up --build
