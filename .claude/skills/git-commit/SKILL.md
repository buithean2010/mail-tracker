---
description: Commit và push sau khi hoàn thành một task trong wbs/tasks.md, theo cú pháp task_id: summary
when_to_use: Khi user hoàn thành một WBS task và muốn commit + push, hoặc hỏi về cách commit trong dự án này
allowed-tools: Bash(git add *), Bash(git commit *), Bash(git push *), Bash(git status *), Bash(git diff *), Read(wbs/tasks.md)
argument-hint: "<task_id> <summary>"
---

# Git Commit — WBS Task

## Cú pháp commit message

```
<task_id>: <summary>

Co-Authored-By: WOZCODE <contact@withwoz.com>
```

**Ví dụ:**
```
P01_03: add docker-compose.yml for local dev environment

Co-Authored-By: WOZCODE <contact@withwoz.com>
```

## Quy trình

```bash
# 1. Kiểm tra thay đổi
git status
git diff --staged

# 2. Stage files liên quan đến task (không dùng git add -A nếu có file nhạy cảm)
git add <files...>

# 3. Commit với đúng cú pháp
git commit -m "$(cat <<'EOF'
$ARGUMENTS

Co-Authored-By: WOZCODE <contact@withwoz.com>
EOF
)"

# 4. Push
git push
```

## Sau khi commit

Cập nhật status task trong `wbs/tasks.md` từ `in-progress` → `done`.

## Lưu ý

- `task_id` lấy từ `wbs/tasks.md` (ví dụ: `P01_03`, `P03_07`)
- Summary ngắn gọn, tiếng Anh, mô tả **what** đã làm
- Không commit `.env`, credentials, hoặc file chứa secret
- Không dùng `--no-verify` trừ khi user yêu cầu rõ ràng
