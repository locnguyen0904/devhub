# 06 — Design System

Nền tảng: **Tailwind CSS v4.3.3** (cấu hình CSS-first bằng `@theme`) + **shadcn/ui**.

Hướng thị giác đã chốt: **thân thiện, nhiều màu** — thẻ bài viết bo góc, có ảnh bìa, tag phân màu theo chủ đề, hiển thị đủ số liệu tương tác. Màu nhấn **indigo/tím**. **Light và dark mode làm song song ngay từ Phase 0.**

## 1. Nguyên tắc

1. **Component không bao giờ viết màu nguyên thuỷ.** Không `bg-white`, không `text-slate-600`. Chỉ dùng token ngữ nghĩa: `bg-surface`, `text-muted`. Đây là điều kiện để dark mode gần như miễn phí — đổi giá trị token, không đụng component.
2. **Mọi cặp màu chữ/nền phải đạt WCAG AA** (4.5:1 văn bản thường, 3:1 văn bản ≥18px hoặc ≥14px đậm). Số đo cụ thể ở §9, kiểm chứng được bằng script chứ không phải cảm tính.
3. **Màu không bao giờ là kênh thông tin duy nhất.** Tag phải đọc được tên, không chỉ phân biệt bằng màu. Trạng thái lỗi phải có chữ, không chỉ viền đỏ.
4. **Thang giá trị hữu hạn.** Khoảng cách, bo góc, cỡ chữ lấy từ thang cho sẵn. Không có `padding: 13px`.

---

## 2. Kiến trúc token

Hai tầng, đừng trộn lẫn:

```
Primitive  →  slate-600, indigo-500, rose-100     (giá trị thô, KHÔNG dùng trong component)
Semantic   →  text-muted, bg-accent, border-subtle (vai trò, đây mới là thứ component dùng)
```

Token ngữ nghĩa khai báo bằng CSS variable ở `:root` và `.dark`, rồi ánh xạ vào Tailwind qua `@theme inline`:

```css
/* frontend/src/app/theme.css */
@import "tailwindcss";

@custom-variant dark (&:where(.dark, .dark *));

:root {
  --surface:          #FFFFFF;   /* nền trang */
  --surface-raised:   #F8FAFC;   /* nền thẻ, panel */
  --surface-sunken:   #F1F5F9;   /* nền code block, input */
  --text-primary:     #0F172A;
  --text-muted:       #475569;
  --text-subtle:      #64748B;
  --border-subtle:    #E2E8F0;
  --border-strong:    #CBD5E1;
  --accent:           #4F46E5;   /* indigo-600 */
  --accent-hover:     #4338CA;
  --accent-fg:        #FFFFFF;   /* chữ nằm TRÊN nền accent */
  --accent-subtle:    #EEF2FF;   /* nền nhạt cho trạng thái active */
}

.dark {
  --surface:          #0F172A;
  --surface-raised:   #1E293B;
  --surface-sunken:   #020617;
  --text-primary:     #F1F5F9;
  --text-muted:       #94A3B8;
  --text-subtle:      #64748B;
  --border-subtle:    #1E293B;
  --border-strong:    #334155;
  --accent:           #818CF8;   /* indigo-400 — sáng hơn để nổi trên nền tối */
  --accent-hover:     #A5B4FC;
  --accent-fg:        #020617;   /* nền accent sáng nên chữ phải tối */
  --accent-subtle:    #1E1B4B;
}

@theme inline {
  --color-surface:        var(--surface);
  --color-surface-raised: var(--surface-raised);
  --color-surface-sunken: var(--surface-sunken);
  --color-text-primary:   var(--text-primary);
  --color-text-muted:     var(--text-muted);
  --color-text-subtle:    var(--text-subtle);
  --color-border-subtle:  var(--border-subtle);
  --color-border-strong:  var(--border-strong);
  --color-accent:         var(--accent);
  --color-accent-hover:   var(--accent-hover);
  --color-accent-fg:      var(--accent-fg);
  --color-accent-subtle:  var(--accent-subtle);
}
```

Điểm dễ bỏ sót: **`--accent-fg` đảo ngược giữa hai chế độ.** Nút chính ở light mode là chữ trắng trên nền indigo đậm; ở dark mode là chữ gần đen trên nền indigo sáng. Ai quen hardcode `text-white` cho nút sẽ tạo ra nút không đọc nổi ở dark mode.

---

## 3. Bảng màu nền tảng

**Trung tính** — thang `slate` của Tailwind. Hơi ngả lạnh, đứng cạnh indigo dễ chịu hơn gray thuần.

| Bậc | Hex | Vai trò |
|---|---|---|
| 50 | `#F8FAFC` | nền nâng (light) |
| 100 | `#F1F5F9` | nền chìm (light) · chữ chính (dark) |
| 200 | `#E2E8F0` | viền nhạt (light) |
| 300 | `#CBD5E1` | viền đậm (light) |
| 400 | `#94A3B8` | chữ phụ (dark) |
| 500 | `#64748B` | chữ mờ (**chỉ light**) |
| — | `#7987A1` | chữ mờ (dark) — xem ghi chú bên dưới |
| 600 | `#475569` | chữ phụ (light) |
| 700 | `#334155` | viền đậm (dark) |
| 800 | `#1E293B` | nền nâng (dark) · viền nhạt (dark) |
| 900 | `#0F172A` | chữ chính (light) · nền trang (dark) |
| 950 | `#020617` | nền chìm (dark) |

> **`text-subtle` ở dark mode không dùng được `slate-500`.** Bản thiết kế đầu tiên dùng chung `#64748B` cho cả hai chế độ; script kiểm tương phản bác bỏ — trên nền `#0F172A` nó chỉ đạt **3.75**, dưới ngưỡng AA. Giá trị thay thế `#7987A1` đạt **4.92** mà vẫn mờ hơn `text-muted` (6.96), nên thứ bậc thị giác được giữ nguyên. Đây đúng là loại lỗi mà mắt thường không bắt được.

**Nhấn** — thang `indigo`.

| Bậc | Hex | Dùng ở |
|---|---|---|
| 400 | `#818CF8` | accent ở dark mode |
| 500 | `#6366F1` | logo, gradient trang trí |
| 600 | `#4F46E5` | accent ở light mode |
| 700 | `#4338CA` | accent hover ở light mode |

**Ngữ nghĩa trạng thái** — dùng `emerald` (thành công), `amber` (cảnh báo), `rose` (lỗi). Cùng quy tắc: bậc 600/700 ở light, bậc 400 ở dark.

---

## 4. Màu tag

Phong cách đã chọn yêu cầu tag phân màu. Nhưng tag do người dùng tự tạo, nên **không thể để mỗi tag một mã hex tuỳ ý** — hex nào đọc được trên nền trắng thì gần như chắc chắn không đọc được trên nền `#0F172A`.

Giải pháp: **bảng 8 sắc cố định**, mỗi sắc là một *cặp* giá trị light/dark đã kiểm tương phản.

| Khoá | Chữ (light) | Nền (light) | Chữ (dark) | Nền (dark) |
|---|---|---|---|---|
| `blue` | `#1D4ED8` | `#DBEAFE` | `#93C5FD` | `#172554` |
| `violet` | `#6D28D9` | `#EDE9FE` | `#C4B5FD` | `#2E1065` |
| `emerald` | `#047857` | `#D1FAE5` | `#6EE7B7` | `#022C22` |
| `amber` | `#B45309` | `#FEF3C7` | `#FCD34D` | `#451A03` |
| `rose` | `#BE123C` | `#FFE4E6` | `#FDA4AF` | `#4C0519` |
| `cyan` | `#0E7490` | `#CFFAFE` | `#67E8F9` | `#083344` |
| `orange` | `#C2410C` | `#FFEDD5` | `#FDBA74` | `#431407` |
| `teal` | `#0F766E` | `#CCFBF1` | `#5EEAD4` | `#042F2E` |

Gán sắc cho tag:

1. Nếu `tags.color_key` có giá trị → dùng nó (admin đặt tay cho tag phổ biến: `go` → `cyan`, `typescript` → `blue`).
2. Nếu `NULL` → `hash(tag.name) % 8`. Ổn định giữa các lần render vì chỉ phụ thuộc tên.

> **Đã kéo theo một sửa đổi ở data model.** `docs/02-data-model.md §5` ban đầu khai báo `color_hex TEXT CHECK (color_hex ~ '^#[0-9a-fA-F]{6}$')` — một mã hex không thể phục vụ cả hai chế độ sáng/tối. Đã đổi thành `color_key` với `CHECK (color_key IN (...))` theo đúng 8 khoá trên, và `docs/03-api.md` đổi theo. Phát hiện lúc còn là tài liệu nên miễn phí; phát hiện sau khi có dữ liệu thì tốn một migration chuyển đổi.

---

## 5. Chữ

**Font**

```css
--font-sans: "Inter var", ui-sans-serif, system-ui, -apple-system, "Segoe UI", sans-serif;
--font-mono: "JetBrains Mono", ui-monospace, "SF Mono", Menlo, monospace;
```

Font tải từ **self-host**, không từ Google Fonts CDN — vừa nhanh hơn (bớt một kết nối DNS/TLS), vừa tránh chuyện gửi IP người đọc sang bên thứ ba.

**Thang cỡ chữ**

| Token | Cỡ | Line-height | Dùng cho |
|---|---|---|---|
| `text-display` | 36px | 1.15 | tiêu đề trang bài viết |
| `text-h1` | 30px | 1.2 | tiêu đề trang |
| `text-h2` | 24px | 1.3 | mục lớn trong bài |
| `text-h3` | 20px | 1.4 | tiêu đề thẻ bài viết |
| `text-body` | 16px | 1.6 | giao diện chung |
| `text-reading` | **18px** | **1.75** | **thân bài viết** |
| `text-sm` | 14px | 1.5 | metadata, nhãn |
| `text-xs` | 12px | 1.4 | tag, timestamp |

**Thân bài viết là ngoại lệ có chủ đích.** 18px với line-height 1.75 và độ rộng dòng giới hạn `max-w-[68ch]` — đây là sản phẩm để đọc bài kỹ thuật dài, không phải dashboard. Dòng quá dài khiến mắt lạc hàng khi xuống dòng.

---

## 6. Khoảng cách, bo góc, đổ bóng

**Khoảng cách** — bội số của 4px, dùng thang mặc định của Tailwind. Ba khoảng cách bố cục cố định:

| Token | Giá trị | Dùng |
|---|---|---|
| `--gap-card` | 16px | padding trong thẻ |
| `--gap-section` | 32px | giữa các khối trên trang |
| `--gap-page` | 24px / 48px | lề trang (mobile / desktop) |

**Bo góc** — phong cách thân thiện nên bo rộng tay:

| Token | Giá trị | Dùng |
|---|---|---|
| `--radius-tag` | 6px | chip tag |
| `--radius-control` | 8px | nút, input |
| `--radius-card` | 12px | thẻ bài viết, panel |
| `--radius-image` | 12px | ảnh bìa |
| `--radius-full` | 9999px | avatar |

**Đổ bóng** — chỉ dùng ở light mode:

```css
--shadow-card:  0 1px 2px rgb(15 23 42 / 0.04), 0 2px 8px rgb(15 23 42 / 0.06);
--shadow-popup: 0 4px 16px rgb(15 23 42 / 0.10), 0 8px 32px rgb(15 23 42 / 0.08);
```

Ở dark mode, bóng đen trên nền đen là vô hình. **Phân tầng bằng độ sáng nền và viền, không bằng bóng**: `.dark` đặt `--shadow-card: none` và thẻ dùng `bg-surface-raised` (`#1E293B`) nổi trên `bg-surface` (`#0F172A`), cộng viền `border-subtle`. Bê nguyên bóng từ light sang dark là lỗi rất hay gặp.

---

## 7. Cơ chế dark mode

- Class `.dark` đặt trên `<html>`. Tailwind v4 khai báo qua `@custom-variant` (§2).
- Ba trạng thái: `light`, `dark`, `system`. Mặc định `system`.
- Lựa chọn lưu ở `localStorage` — **và chỉ được đọc/ghi trong `shared/`**, theo `CLAUDE.md §4`.
- **Chống nháy trắng:** script đồng bộ đặt class trước khi React mount, nhúng thẳng vào `index.html`. Nếu để React set trong `useEffect` thì người dùng dark mode sẽ thấy một khung hình trắng loé lên mỗi lần tải trang.

```html
<script>
  // Chạy trước khi paint: đọc lựa chọn đã lưu, ngã về sở thích hệ điều hành.
  // Đặt trong useEffect sẽ gây nháy trắng một khung hình ở dark mode.
  (function () {
    var saved = localStorage.getItem("theme");
    var dark = saved === "dark" || (saved !== "light" &&
      window.matchMedia("(prefers-color-scheme: dark)").matches);
    document.documentElement.classList.toggle("dark", dark);
  })();
</script>
```

- Thêm `<meta name="color-scheme" content="light dark">` để thanh cuộn và ô nhập liệu mặc định của trình duyệt cũng đổi theo.

---

## 8. Code block

`chroma` chạy ở chế độ `WithClasses(true)` — sinh ra `<span class="k">`, `<span class="s">`… chứ không nhúng style inline. Nhờ vậy màu do CSS quyết định và đổi được theo chế độ sáng/tối; nếu để chroma nhúng inline thì code block sẽ kẹt một theme duy nhất.

Chỉ cần tô 6 nhóm token — nhiều hơn là rối chứ không rõ hơn:

| Lớp | Nhóm | Light | Dark |
|---|---|---|---|
| `.k` `.kd` | từ khoá | `#6D28D9` | `#C4B5FD` |
| `.s` `.s1` `.s2` | chuỗi | `#047857` | `#6EE7B7` |
| `.c` `.cm` `.c1` | chú thích | `#64748B` | `#94A3B8` |
| `.nf` | tên hàm | `#1D4ED8` | `#93C5FD` |
| `.m` `.mi` | số | `#C2410C` | `#FDBA74` |
| còn lại | chữ thường | `--text-primary` | `--text-primary` |

Nền code block dùng `--surface-sunken`. **Màu chú thích vẫn phải đạt AA** — chú thích xám nhạt đến mức không đọc nổi là lỗi phổ biến của nhiều theme code.

---

## 9. Khả năng tiếp cận

Toàn bộ số dưới đây do script ở §11 đo, không phải ước lượng.

**Light mode (nền `#FFFFFF`)**

| Cặp màu | Tỉ lệ | Ngưỡng |
|---|---|---|
| chữ chính `slate-900` | **17.85** | 4.5 |
| chữ phụ `slate-600` | **7.58** | 4.5 |
| chữ mờ `slate-500` | **4.76** | 4.5 |
| link `indigo-600` | **6.29** | 4.5 |
| nút chính (trắng trên `indigo-600`) | **6.29** | 4.5 |

**Dark mode (nền `#0F172A`)**

| Cặp màu | Tỉ lệ | Ngưỡng |
|---|---|---|
| chữ chính `slate-100` | **16.30** | 4.5 |
| chữ phụ `slate-400` | **6.96** | 4.5 |
| chữ mờ `#7987A1` | **4.92** | 4.5 |
| link `indigo-400` | **5.98** | 4.5 |
| nút chính (`slate-950` trên `indigo-400`) | **6.76** | 4.5 |
| chữ phụ trên thẻ (`slate-400`/`slate-800`) | **5.71** | 4.5 |

**Tag** — tất cả 8 sắc đạt AA ở cả hai chế độ. Thấp nhất: `amber` 4.51 và `orange` 4.52 ở light mode. **Sát ngưỡng — đừng chỉnh hai sắc này sáng hơn**, chỉnh một chút là rớt chuẩn. Ở dark mode dư dả hơn nhiều (8.15–10.39).

**Quy tắc khác**

- Focus ring: `2px` màu `--accent` + `2px` offset, **không bao giờ `outline: none`** nếu chưa thay bằng chỉ báo khác. Dùng `:focus-visible` để chuột không kích hoạt ring.
- Vùng bấm tối thiểu 44×44px trên thiết bị cảm ứng, kể cả khi icon nhỏ hơn.
- Tôn trọng `prefers-reduced-motion: reduce` — tắt hiệu ứng chuyển cảnh, giữ lại thay đổi trạng thái.
- Mọi ảnh phải có `alt`. Ảnh trang trí thì `alt=""` để trình đọc màn hình bỏ qua.
- Không dùng `outline` hay màu để truyền tải thông tin duy nhất (§1.3).

---

## 10. Quy ước component

- **Không sửa file trong `shared/ui/`** (shadcn sinh ra) trừ khi để thay màu nguyên thuỷ bằng token ngữ nghĩa. Cần biến thể mới thì bọc bên ngoài, để còn `npx shadcn add` cập nhật được.
- Biến thể của component khai báo bằng `cva` (class-variance-authority), không bằng chuỗi `className` nối tay.
- Mỗi trạng thái của màn hình phải có thiết kế: **loading (skeleton) · empty · error · có dữ liệu**. Trạng thái rỗng phải nói được bước tiếp theo, không chỉ hiện chữ "Không có dữ liệu".
- Skeleton phải đúng kích thước nội dung thật, nếu không trang sẽ giật khi dữ liệu về (CLS).

---

## 11. Kiểm chứng

Script đo tương phản đặt tại `frontend/scripts/check-contrast.mjs`, chạy trong CI cùng `pnpm lint`. Đổi bất kỳ token màu nào mà rớt AA thì CI đỏ — ràng buộc này rẻ, và nó giữ cho mục §9 luôn đúng thay vì thành tài liệu chết.

```js
const hex = h => [1, 3, 5].map(i => parseInt(h.slice(i, i + 2), 16) / 255);
const lin = c => (c <= 0.03928 ? c / 12.92 : Math.pow((c + 0.055) / 1.055, 2.4));
const lum = h => { const [r, g, b] = hex(h).map(lin); return 0.2126 * r + 0.7152 * g + 0.0722 * b; };

// WCAG 2.1 §1.4.3. Nguồn: https://www.w3.org/TR/WCAG21/#contrast-minimum
export const ratio = (fg, bg) => {
  const [hi, lo] = [lum(fg), lum(bg)].sort((a, b) => b - a);
  return (hi + 0.05) / (lo + 0.05);
};
```
