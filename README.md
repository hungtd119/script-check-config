# Script Validate Config Activity

Script Go để validate file config `activityopen.csv` theo 14 quy tắc kiểm tra định nghĩa trong `checklist.csv`.

---

## Yêu cầu

- [Go](https://golang.org/dl/) >= 1.21

---

## Cài đặt

```bash
git clone <repo-url>
cd script-check-config
```

---

## Cách chạy

### Validate file mặc định (`activityopen.csv` trong thư mục hiện tại)

```bash
go run main.go
```

### Validate file khác

```bash
go run main.go path/to/activityopen.csv
```

### Build thành binary rồi chạy

```bash
go build -o validator
./validator
# hoặc chỉ định file
./validator path/to/activityopen.csv
```

---

## Output

```
╔══════════════════════════════════════════════════════════════════════╗
║  VALIDATION REPORT: activityopen.csv                                 ║
╚══════════════════════════════════════════════════════════════════════╝

  ✅  Rule 1   Ô trống có dấu cách
  ✅  Rule 2   Trùng ID
  ✅  Rule 3   Khoảng serverID không chính xác
  ❌  Rule 4   Binh Pháp Nghiên Tập – chồng thời gian (±7 ngày)   (1 lỗi)
      └─  Line 363   (id=2419  )  activityId=2002 chồng thời gian (±7 ngày) với line 321 (id=2261)
  ...

══════════════════════════════════════════════════════════════════════
Kết quả: 9/14 PASS   |   5/14 FAIL
══════════════════════════════════════════════════════════════════════
```

- **✅ PASS** – Rule không phát hiện lỗi nào.
- **❌ FAIL** – Rule phát hiện lỗi, kèm danh sách chi tiết từng dòng vi phạm.
- Mỗi lỗi hiển thị: số dòng trong file CSV, `id` của dòng đó, và mô tả lỗi cụ thể.

---

## Danh sách 14 Rule

| Rule | Tên | Mô tả ngắn |
|------|-----|------------|
| 1 | Ô trống có dấu cách | Ô rỗng phải là `""`, không chấp nhận khoảng trắng |
| 2 | Trùng ID | `id` phải là số nguyên duy nhất trong toàn file |
| 3 | Khoảng serverID | `server` / `except_server` phải là JSON array hợp lệ, giá trị trong `[0, 9999999999]` |
| 4 | Binh Pháp Nghiên Tập – thời gian | `activityId ∈ {2002, 2166, 2167, 2168}`: không được có 2 instance cùng ID cách nhau dưới 7 ngày |
| 5 | Binh Pháp Nghiên Tập – field | Các field bắt buộc: `area=2`, `acType=159`, `levelVis=55`, ... |
| 6 | Tướng UR – thời gian | `activityId ∈ {7601, 7602, 7603, 7604}`: tương tự Rule 4 |
| 7 | Tướng UR – field | Các field bắt buộc: `area=1`, `acType=165`, `integral=1`, ... |
| 8 | Vũ Khí Chuyên Dụng – thời gian | `activityId ∈ {80102..80106}`: tương tự Rule 4 |
| 9 | Vũ Khí Chuyên Dụng – field | Các field bắt buộc: `area=1`, `acType=167`, `levelVis=26`, ... |
| 10 | serverLimit khi startTimeType=1 | `serverLimit` chỉ được là `0` hoặc `168` |
| 11 | Format datetime | `startTime`, `endTime`, `appearTime`, `disappearTime` phải đúng format `YYYY/MM/DD HH:MM:SS` |
| 12 | Sự kiện song hành | Các cặp `{3510↔3511}`, `{194/355/356↔156}`, `{7002↔299}` hoạt động cùng lúc phải có thời gian giống hệt nhau |
| 13 | startTime = appearTime | `startTime` phải bằng `appearTime`, `endTime` phải bằng `disappearTime` |
| 14 | Quỹ tháng cố định | `activityId ∈ {5013..5025}`: thời gian phải đúng theo từng tháng trong năm, kèm các field cố định |

> Tài liệu chi tiết từng rule: [checklist.md](checklist.md)  
> Tài liệu cấu trúc file config: [activityopen.md](activityopen.md)

---

## Cấu trúc project

```
script-check-config/
├── main.go             # Logic validate (14 rules)
├── go.mod
├── activityopen.csv    # File config cần validate
├── checklist.csv       # Định nghĩa 14 rules
├── activityopen.md     # Mô tả cấu trúc activityopen.csv
└── checklist.md        # Mô tả chi tiết từng rule
```
