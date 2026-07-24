# CLAUDE.md

Quy tắc bắt buộc khi làm việc trên repo này. Đọc [docs/](docs/) để hiểu thiết kế trước khi sửa code.

- Ngôn ngữ: **mọi thứ bên trong file code viết tiếng Anh**; chỉ `docs/`, commit message và trao đổi với người dùng viết tiếng Việt.
  - **Tiếng Anh** — tên định danh, comment, godoc; log message; **chuỗi lỗi trả về client**; **nhãn UI hiển thị cho người dùng cuối**; mô tả API (`Summary`/`Description`/`doc:` của huma) và help text CLI (`make help`). Ranh giới đơn giản: nếu nó nằm trong file `.go`/`.ts`/`.tsx`/`.sql`/`.mjs`/`Makefile`/`.yml`, nó tiếng Anh — không phân biệt ai đọc.
  - **Tiếng Việt** — nội dung trong `docs/`, commit message, và tin nhắn trao đổi.
  - Lý do gộp cả text người dùng vào tiếng Anh: tránh ranh giới "chuỗi này ai đọc" vốn mơ hồ và hay cãi nhau. Khi cần đa ngôn ngữ thì thêm một lớp i18n, không rải tiếng Việt trong code.
- Kiến trúc đã chốt ở `docs/01-architecture.md`. Muốn làm khác thì nêu lý do trước, đừng làm rồi báo sau.

---

## 1. Quy tắc làm việc

**Nêu giả định trước khi code.** Yêu cầu mơ hồ có nhiều cách hiểu thì trình bày các cách, đừng tự chọn im lặng. Chỉ hỏi khi hai cách hiểu dẫn tới hai kết quả khác hẳn nhau; còn lại tự quyết và nói rõ đã quyết gì.

**Viết lượng code tối thiểu giải quyết đúng việc được giao.**

- Không thêm tính năng ngoài yêu cầu.
- Không tạo abstraction cho thứ chỉ dùng một chỗ. Interface ra đời khi có người dùng thứ hai, hoặc khi cần mock để test — không phải "để sau này dễ mở rộng".
- Không xử lý lỗi cho tình huống không thể xảy ra.
- Không thêm tham số cấu hình không ai yêu cầu.

**Sửa đúng chỗ cần sửa.** Không "tiện tay" format lại, đổi tên, hay refactor code xung quanh. Bám theo style sẵn có trong file kể cả khi bạn thích kiểu khác. Thấy code chết không liên quan thì **báo, đừng xoá**. Nhưng nếu thay đổi của bạn làm một import/biến/hàm trở nên thừa, dọn nó — đó là rác của chính bạn.

**Mỗi dòng thay đổi phải truy ngược được về yêu cầu.**

**Định nghĩa tiêu chí xong trước khi bắt đầu.** "Thêm validate" → "viết test cho input sai, làm nó pass". "Sửa bug" → "viết test tái hiện bug, làm nó pass". Việc nhiều bước thì nêu kế hoạch ngắn kèm cách kiểm chứng từng bước.

**Không báo xong khi chưa chạy được lệnh kiểm chứng ở §6.** Test fail thì nói thẳng là fail kèm output.

---

## 2. Comment

Quy tắc gốc: **comment giải thích TẠI SAO, code tự nói CÁI GÌ.** Cần comment để hiểu code đang làm gì → sửa code, đừng thêm comment.

### Bắt buộc phải có comment

1. **Quyết định phản trực giác hoặc đánh đổi có chủ đích** — người đọc sau sẽ tưởng là bug và "sửa" nó.
2. **Workaround cho bug thư viện bên thứ ba** — kèm link issue, để còn biết khi nào gỡ được.
3. **Bất biến mà compiler không giữ giúp** — "hàm này phải gọi trong transaction", "map này chỉ đọc sau khi khởi tạo".
4. **Công thức, thuật toán, hằng số ma thuật** — kèm nguồn hoặc cách suy ra.
5. **Godoc cho mọi identifier exported trong Go** — là **câu hoàn chỉnh**, bắt đầu bằng chính tên đó, viết hoa chữ đầu, kết thúc bằng dấu chấm. (Đây là quy ước của `go doc`, không phải sở thích.)

```go
// Good — giải thích lý do, thứ code không nói được
// Sanitize AFTER rendering: markdown allows raw HTML, so sanitizing the
// source would both miss injected tags and corrupt valid markdown syntax.
html := policy.Sanitize(rendered)

// Good — bất biến compiler không giữ
// Caller must hold a transaction: publishing writes posts and post_tags,
// and a partial write leaves the post unreachable.
func (r *repository) publishTx(ctx context.Context, tx pgx.Tx, ...)

// Good — hằng số ma thuật
// Hacker News ranking: comments weigh 2x reactions because they cost more
// effort, so they signal quality more strongly. See docs/02-data-model.md §8.
const commentWeight = 2
```

### Cấm

```go
// Bad — lặp lại y hệt code
// increment the counter
counter++

// Bad — comment chia section, dấu hiệu file nên tách ra
// ---------- helpers ----------

// Bad — code bị comment out. Git đã nhớ giùm rồi, xoá đi.
// oldValue := compute(x)

// Bad — TODO không có chủ và không có hạn
// TODO: fix this later

// Bad — changelog trong code. Đó là việc của git log.
// Updated 2026-07-24: added tag support

// Bad — godoc chỉ chép lại tên hàm
// GetPostByID gets a post by ID.
func GetPostByID(...)
```

TODO chỉ chấp nhận khi có ngữ cảnh và điểm dừng: `// TODO(loc): switch to Meilisearch when p95 exceeds 500ms — see docs/04-roadmap.md`.

---

## 3. Go

### Lỗi

- Bọc lỗi kèm ngữ cảnh: `fmt.Errorf("publish post %s: %w", id, err)`. Luôn `%w`, không `%v`.
- Thông điệp lỗi **viết thường, không dấu chấm cuối** — nó sẽ bị nối vào chuỗi lỗi khác.
- So sánh bằng `errors.Is` / `errors.As`, không so sánh chuỗi.
- **Log một lần ở biên** (handler / middleware). Tầng dưới trả lỗi lên, không vừa log vừa return — làm vậy một lỗi hiện 4 lần trong log.
- `panic` chỉ dùng ở `main`/`init` khi thiếu cấu hình bắt buộc. Không panic trong handler.
- **Không nuốt lỗi bằng `_`.** Nếu thật sự bỏ qua được thì phải có comment nói vì sao bỏ qua an toàn.
- Không báo lỗi bằng giá trị trong dải hợp lệ (`-1`, chuỗi rỗng). Trả thêm `error` hoặc `bool`.

### Kiểu và hàm

- `context.Context` là tham số **đầu tiên**, tên `ctx`. Không nhét context vào struct.
- Trả về sớm, không `else` sau `return`. Giữ luồng chính lệch trái nhất.
- Không dùng named return trừ khi `defer` cần sửa giá trị trả về.
- Không `any`/`interface{}` khi biết kiểu cụ thể hoặc dùng được generic.
- Pointer receiver hay value receiver thì nhất quán trong cùng một kiểu.
- Không viết `init()`. Khởi tạo tường minh trong `main`.
- Mọi goroutine phải có đường dừng qua `ctx.Done()`. Không sinh goroutine mà không biết ai đóng nó.
- Type assertion luôn dùng comma-ok: `v, ok := x.(T)`. Assertion trần làm panic lúc chạy.
- Khẳng định implement interface lúc biên dịch: `var _ Service = (*service)(nil)`. Sai chữ ký thì lỗi ngay tại file cài đặt, không phải ở chỗ gọi.

### Bảo mật

- **Mọi giá trị bí mật sinh bằng `crypto/rand`, không bao giờ `math/rand`** — OAuth `state`, refresh token, salt. `math/rand` đoán được, dùng nhầm là mất tài khoản người dùng.
- Refresh token chỉ lưu hash trong DB, không lưu bản thô (`docs/02-data-model.md §3`).

### Đặt tên

- Không lặp tên package: `post.Service`, không phải `post.PostService`.
- Không đặt tên package là `util`, `common`, `helper`, `misc` — tên không nói gì thì package sẽ hút mọi thứ vào.
- Interface nhỏ đặt theo hành vi: `UserFinder`, `TokenStore`.
- Biến sống ngắn thì tên ngắn (`i`, `p` trong vòng lặp); biến sống dài thì tên đủ nghĩa.
- Viết tắt giữ nguyên hoa/thường: `postID`, `httpClient`, `userURL` — không `postId`, `HttpClient`.
- Receiver 1–2 ký tự, viết tắt theo tên kiểu, nhất quán trong cùng kiểu: `func (s *service)`. Không `this`, không `self`.
- Không dùng tiền tố `Get`: `post.Tags()`, không `post.GetTags()`.

### Database

- **Không viết chuỗi SQL trong file `.go`.** Query nằm ở `db/queries/*.sql`, sinh code bằng `make sqlc`.
- **Không trả struct do sqlc sinh ra thẳng ra JSON.** Chuyển sang domain model ở repository, sang DTO ở handler.
- Ghi nhiều bảng thì phải trong một transaction.
- Query trả về danh sách luôn có `LIMIT`.

### Test

- **Bắt buộc test cho `service.go`** (nghiệp vụ) và `repository.go` (SQL thật qua testcontainers). Handler và UI thì tùy.
- Test repository chạy trên **Postgres thật**, không SQLite — schema này dùng `tsvector`, partial index, `ON CONFLICT`, SQLite không có.
- Table-driven test, mỗi case có tên đọc được: `"rejects publish when actor is not author"`.
- Thông điệp fail phải cho biết **input, kết quả thật, kết quả mong đợi**: `Publish(%v) = %v, want %v`. Test fail mà phải mở code ra mới hiểu là test kém.
- **So sánh struct/slice/map bằng `cmp.Diff`**, không `require.Equal`, không `reflect.DeepEqual`. `require.NoError`/`require.Error` thì dùng testify. Lý do phân vai ở `docs/05-go-stack.md`.
- Mock repository bằng interface tự viết tay. Chưa dùng thư viện mock ở quy mô này.
- Test kiểm chứng **hành vi**, không kiểm chứng cách cài đặt. Đổi implementation mà test đỏ trong khi hành vi không đổi là test sai.

---

## 4. TypeScript / React

### Kiểu

- `strict: true`. **Không `any`** — dùng `unknown` rồi thu hẹp kiểu.
- Không ép kiểu bằng `as` và không dùng `!` để bỏ qua null. Ép kiểu là tự nhận "tôi biết rõ hơn compiler" — nếu đúng vậy thì viết type guard.
- **Kiểu của API sinh từ OpenAPI** (`shared/types/api.ts`), không tự tay khai báo lại interface cho response. Tay viết là sớm muộn cũng lệch với backend.
- **`interface` cho hình dạng object; `type` cho union, tuple, alias của primitive.** Theo Google TS Style Guide. Chọn cái nào không quan trọng bằng việc nhất quán — ESLint `consistent-type-definitions` giữ giúp.
- **Không chú thích kiểu cho thứ suy ra được hiển nhiên.** `const name = "x"` chứ không `const name: string = "x"`. Chú thích khi biểu thức phức tạp hoặc khi muốn chốt kiểu ở biên module.
- Callback bỏ qua giá trị trả về thì khai `=> void`, không `=> any` — `any` khiến người khác lỡ tay dùng giá trị đó mà không bị chặn.
- **Viết tắt trong TS viết như từ thường: `postId`, `httpClient`, `userUrl` — ngược với Go (`postID`, `userURL`).** Mỗi ngôn ngữ một quy ước; đừng bê quy ước Go sang TS.

### React

- **Server state dùng TanStack Query. Client state dùng `useState`/Zustand.** Không copy dữ liệu từ query vào store — đó là nguồn gốc của bug dữ liệu cũ.
- **Không dùng `useEffect` để fetch dữ liệu.** Đó là việc của TanStack Query.
- `useEffect` **chỉ** dành cho đồng bộ với hệ thống bên ngoài (DOM, timer, subscription). Bốn trường hợp hay bị dùng sai, theo [react.dev](https://react.dev/learn/you-might-not-need-an-effect):
  - Giá trị dẫn xuất từ props/state → tính thẳng khi render, không `setState` trong effect.
  - Tính toán đắt → `useMemo`, không effect.
  - Reset state khi prop đổi → đổi `key` của component, không effect.
  - Phản ứng với hành động người dùng → viết trong event handler. Effect không biết nút nào vừa được bấm.
- Props khai báo type tường minh. Không dùng `React.FC`.
- Không viết nghiệp vụ trong component. Component lo hiển thị; logic nằm ở hook hoặc hàm thuần test được.
- Danh sách phải có `key` là id ổn định, không dùng index.

### Tổ chức file

- Chia theo tính năng (`features/posts/`), không chia theo loại file (`components/`, `hooks/` toàn cục).
- File component `PascalCase.tsx`, còn lại `kebab-case.ts`.
- **Named export**, không default export (trừ file config bắt buộc).
- **Mọi lời gọi API đi qua `shared/api/client.ts`.** Không `fetch` rải rác trong component.
- Không dùng `window.location`, `document.cookie`, `localStorage` ngoài tầng `shared/`.

---

## 5. Ranh giới không được vượt

Vi phạm những điều này là phá kiến trúc, không phải khác biệt phong cách:

1. **Module không import repository của module khác.** Cần dữ liệu từ module khác thì khai báo interface trong `ports.go` của chính mình (`docs/01-architecture.md §4`).
2. **`service.go` và `repository.go` không import `huma`.** `huma` chỉ được xuất hiện ở `handler.go`.
3. **Markdown phải sanitize sau khi render, trước khi lưu.** Bỏ bước này là mở cửa cho stored XSS — đăng một bài chiếm được phiên của mọi người đọc.
4. **Không dùng offset pagination.** Cursor, khớp với index đã thiết kế.
5. **Không đưa secret vào code hay commit.** Tất cả qua biến môi trường, khai báo trong `config.Config` với tag `required`.
6. **Migration không tự chạy lúc API khởi động.** Là bước deploy riêng.

---

## 6. Trước khi báo xong

Chạy và phải sạch:

```bash
make lint          # golangci-lint
make test          # go test ./...
cd frontend && pnpm typecheck && pnpm lint
```

Đổi API thì chạy thêm `make openapi` để sinh lại type TypeScript, và commit file sinh ra.

Đổi schema thì migration phải có **cả `.up.sql` và `.down.sql`**, và đã thử `make migrate-down` rồi `make migrate-up` lại.

Commit message viết tiếng Việt, dòng đầu ≤ 72 ký tự, mô tả **tại sao** chứ không phải liệt kê file đã sửa.

---

## 7. Nguồn

Các rule trên rút từ những tài liệu này. Khi tranh cãi, tài liệu thắng — nếu bạn thấy chỗ nào trong file này lệch với chúng thì báo.

**Go**
- [Go Code Review Comments](https://go.dev/wiki/CodeReviewComments) — chuẩn của chính team Go
- [Google Go Style Guide](https://google.github.io/styleguide/go/decisions.html)
- [Uber Go Style Guide](https://github.com/uber-go/guide/blob/master/style.md)

**TypeScript / React**
- [Google TypeScript Style Guide](https://google.github.io/styleguide/tsguide.html)
- [TypeScript Do's and Don'ts](https://www.typescriptlang.org/docs/handbook/declaration-files/do-s-and-don-ts.html)
- [react.dev — You Might Not Need an Effect](https://react.dev/learn/you-might-not-need-an-effect)

**Nguyên tắc phân chia:** rule nào linter kiểm được thì thuộc về `.golangci.yml` / `eslint.config.js`, **không** chép vào file này. File này chỉ giữ thứ máy không kiểm nổi. Hai chỗ định nghĩa cùng một sự thật là hai chỗ sẽ lệch nhau.
