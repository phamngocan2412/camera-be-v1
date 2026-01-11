## 2024-05-23 - [Database] Skip Redundant Updates
**Vấn đề:** Hàm `UpdateProfile` trong `UserService` luôn gọi `repo.Update` (kích hoạt SQL `UPDATE` và cập nhật `updated_at`) ngay cả khi request không chứa thay đổi nào hoặc dữ liệu trùng khớp với hiện tại. Điều này gây lãng phí I/O database.
**Giải pháp:** Thêm biến cờ `updated` để theo dõi xem có trường nào thực sự thay đổi không. Chỉ gọi `repo.Update` khi `updated == true`.
