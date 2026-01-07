## 2024-05-23 - Tối ưu Error Handling (Sentinel Errors)
**Vấn đề:**
- Code cũ sử dụng `err.Error() == "string"` để kiểm tra lỗi nghiệp vụ (như Email đã tồn tại, mật khẩu cũ sai).
- Cách này chậm (so sánh chuỗi), dễ lỗi (typo trong string), và khó bảo trì.
- Vi phạm nguyên tắc "Clean Code" trong Go.

**Giải pháp:**
- Định nghĩa các biến lỗi (Sentinel Errors) exported trong `service` package: `ErrEmailExists`, `ErrOldPasswordIncorrect`.
- Sử dụng `errors.Is()` để kiểm tra lỗi.
- **Hiệu năng:** So sánh con trỏ (pointer comparison) nhanh hơn so sánh chuỗi.
- **Maintainability:** Type-safe, refactoring dễ dàng.
