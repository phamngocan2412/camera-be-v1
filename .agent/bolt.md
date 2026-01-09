## 2024-05-22 - Skip Unnecessary DB Updates
**Vấn đề:** Hàm `UpdateProfile` trong `UserService` luôn gọi `repo.Update(user)` ngay cả khi dữ liệu đầu vào (Request) không có thay đổi hoặc rỗng. Việc này gây ra một thao tác ghi xuống Database (I/O) lãng phí, kích hoạt cập nhật cột `updated_at` và các overhead của GORM/DB Transaction không cần thiết.
**Giải pháp:** Thêm logic kiểm tra cờ `isUpdated`. Chỉ gọi `repo.Update` khi thực sự có thay đổi dữ liệu trong memory. Giúp giảm tải cho Database và API phản hồi nhanh hơn trong trường hợp user gửi request thừa.
