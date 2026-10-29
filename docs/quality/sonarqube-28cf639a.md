# SonarQube Cloud issue review — 28cf639a

This register maps all 77 issue keys from the supplied SonarQube Cloud export for analysis commit `28cf639a`. It records the finding's original path and line, plus the disposition checked against `fix/sonar-findings`.

The cloud snapshot is unchanged until a new analysis runs. The register does not claim that SonarQube has closed these issues. Four password-rule findings point to translated explanatory copy and remain false-positive candidates; the rule was not disabled.

| Issue key | Rule | Severity | Reported location | Disposition | Evidence / conclusion |
|---|---|---|---|---|---|
| AaFt8Zic7GRUxAPz9K5I | Web:S9379 | MAJOR | frontend/src/views/ShareView.vue:179 | 已修正或核對，待重掃 | 移除分享密碼欄位 autofocus；保留表單錯誤時的明確焦點管理。 |
| AaFt8Zic7GRUxAPz9K5J | typescript:S9011 | MAJOR | frontend/src/views/ShareView.vue:179 | 已修正或核對，待重掃 | 核對目前 Vue template 的按鈕皆明確指定 button 或 submit。 |
| AaFt8ZiT7GRUxAPz9K4t | Web:S6840 | MAJOR | frontend/src/views/ShareManagerView.vue:164 | 已修正或核對，待重掃 | 已核對登入、註冊、密碼與 Email 欄位的 autocomplete 語意。 |
| AaFt8ZiT7GRUxAPz9K4u | Web:S6853 | MAJOR | frontend/src/views/ShareManagerView.vue:164 | 已修正或核對，待重掃 | 已修正標籤無法指向按鈕的自訂選擇器，改用可存取名稱。 |
| AaFt8ZiT7GRUxAPz9K4v | Web:InputWithoutLabelCheck | MAJOR | frontend/src/views/ShareManagerView.vue:164 | 已修正或核對，待重掃 | 已核對欄位 id 與標籤關係；自訂選擇器按鈕有可存取名稱。 |
| AaFt8ZiT7GRUxAPz9K4w | Web:InputWithoutLabelCheck | MAJOR | frontend/src/views/ShareManagerView.vue:164 | 已修正或核對，待重掃 | 已核對欄位 id 與標籤關係；自訂選擇器按鈕有可存取名稱。 |
| AaFt8ZiT7GRUxAPz9K4x | Web:InputWithoutLabelCheck | MAJOR | frontend/src/views/ShareManagerView.vue:164 | 已修正或核對，待重掃 | 已核對欄位 id 與標籤關係；自訂選擇器按鈕有可存取名稱。 |
| AaFt8ZiW7GRUxAPz9K42 | Web:InputWithoutLabelCheck | MAJOR | frontend/src/views/ProfileView.vue:116 | 已修正或核對，待重掃 | 已核對欄位 id 與標籤關係；自訂選擇器按鈕有可存取名稱。 |
| AaFt8ZiW7GRUxAPz9K43 | Web:InputWithoutLabelCheck | MAJOR | frontend/src/views/ProfileView.vue:116 | 已修正或核對，待重掃 | 已核對欄位 id 與標籤關係；自訂選擇器按鈕有可存取名稱。 |
| AaFt8ZiW7GRUxAPz9K44 | Web:InputWithoutLabelCheck | MAJOR | frontend/src/views/ProfileView.vue:117 | 已修正或核對，待重掃 | 已核對欄位 id 與標籤關係；自訂選擇器按鈕有可存取名稱。 |
| AaFt8Zii7GRUxAPz9K5Z | typescript:S2068 | MAJOR | frontend/src/locales/zh-TW.ts:41 | 誤報候選 | 翻譯文案提及密碼規則，屬誤報候選；保留文案，不停用規則，等待重掃人工確認。 |
| AaFt8Zii7GRUxAPz9K5a | typescript:S2068 | MAJOR | frontend/src/locales/zh-TW.ts:41 | 誤報候選 | 翻譯文案提及密碼規則，屬誤報候選；保留文案，不停用規則，等待重掃人工確認。 |
| AaFt8ZiT7GRUxAPz9K4y | typescript:S9011 | MAJOR | frontend/src/views/ShareManagerView.vue:167 | 已修正或核對，待重掃 | 核對目前 Vue template 的按鈕皆明確指定 button 或 submit。 |
| AaFt8Zii7GRUxAPz9K5Y | typescript:S2068 | MAJOR | frontend/src/locales/zh-TW.ts:37 | 誤報候選 | 翻譯文案提及密碼規則，屬誤報候選；保留文案，不停用規則，等待重掃人工確認。 |
| AaFt8ZgW7GRUxAPz9K4e | go:S2092 | MINOR | backend/internal/transport/httpapi/api.go:233 | 已修正或核對，待重掃 | Cookie 在直接 TLS 或 PocketBase Application URL 為 HTTPS 時設 Secure；明確 HTTP 相容模式保留。 |
| AaFt8ZiV7GRUxAPz9K4z | typescript:S9011 | MAJOR | frontend/src/views/DashboardView.vue:125 | 已修正或核對，待重掃 | 核對目前 Vue template 的按鈕皆明確指定 button 或 submit。 |
| AaFt8ZiX7GRUxAPz9K4- | typescript:S9011 | MAJOR | frontend/src/views/GroupsView.vue:42 | 已修正或核對，待重掃 | 核對目前 Vue template 的按鈕皆明確指定 button 或 submit。 |
| AaFt8ZiX7GRUxAPz9K46 | Web:S6853 | MAJOR | frontend/src/views/GroupsView.vue:42 | 已修正或核對，待重掃 | 已修正標籤無法指向按鈕的自訂選擇器，改用可存取名稱。 |
| AaFt8ZiX7GRUxAPz9K47 | Web:S6853 | MAJOR | frontend/src/views/GroupsView.vue:42 | 已修正或核對，待重掃 | 已修正標籤無法指向按鈕的自訂選擇器，改用可存取名稱。 |
| AaFt8Zig7GRUxAPz9K5S | Web:S6840 | MAJOR | frontend/src/views/SetupView.vue:20 | 已修正或核對，待重掃 | 已核對登入、註冊、密碼與 Email 欄位的 autocomplete 語意。 |
| AaFt8Zig7GRUxAPz9K5T | Web:S6840 | MAJOR | frontend/src/views/SetupView.vue:20 | 已修正或核對，待重掃 | 已核對登入、註冊、密碼與 Email 欄位的 autocomplete 語意。 |
| AaFt8Zig7GRUxAPz9K5U | Web:S6853 | MAJOR | frontend/src/views/SetupView.vue:20 | 已修正或核對，待重掃 | 已修正標籤無法指向按鈕的自訂選擇器，改用可存取名稱。 |
| AaFt8Zig7GRUxAPz9K5V | Web:S6853 | MAJOR | frontend/src/views/SetupView.vue:20 | 已修正或核對，待重掃 | 已修正標籤無法指向按鈕的自訂選擇器，改用可存取名稱。 |
| AaFt8Zig7GRUxAPz9K5W | typescript:S9011 | MAJOR | frontend/src/views/SetupView.vue:20 | 已修正或核對，待重掃 | 核對目前 Vue template 的按鈕皆明確指定 button 或 submit。 |
| AaFt8Zii7GRUxAPz9K5X | typescript:S2068 | MAJOR | frontend/src/locales/zh-TW.ts:26 | 誤報候選 | 翻譯文案提及密碼規則，屬誤報候選；保留文案，不停用規則，等待重掃人工確認。 |
| AaFt8Zib7GRUxAPz9K5F | typescript:S9011 | MAJOR | frontend/src/views/MembersView.vue:161 | 已修正或核對，待重掃 | 核對目前 Vue template 的按鈕皆明確指定 button 或 submit。 |
| AaFt8Ziq7GRUxAPz9K5h | githubactions:S6505 | MAJOR | .github/workflows/ci.yml:69 | 已修正或核對，待重掃 | workflow 安裝命令使用 --ignore-scripts。 |
| AaFt8Zip7GRUxAPz9K5e | githubactions:S7637 | MAJOR | .github/workflows/verify.yml:36 | 已修正或核對，待重掃 | workflow action 使用完整 commit SHA。 |
| AaFt8Zip7GRUxAPz9K5f | githubactions:S6505 | MAJOR | .github/workflows/verify.yml:49 | 已修正或核對，待重掃 | workflow 安裝命令使用 --ignore-scripts。 |
| AaFt8ZiL7GRUxAPz9K4k | Web:InputWithoutLabelCheck | MAJOR | frontend/src/components/CategorySelect.vue:21 | 已修正或核對，待重掃 | 已核對欄位 id 與標籤關係；自訂選擇器按鈕有可存取名稱。 |
| AaFt8ZiL7GRUxAPz9K4l | Web:InputWithoutLabelCheck | MAJOR | frontend/src/components/CategorySelect.vue:21 | 已修正或核對，待重掃 | 已核對欄位 id 與標籤關係；自訂選擇器按鈕有可存取名稱。 |
| AaFt8ZiI7GRUxAPz9K4h | Web:InputWithoutLabelCheck | MAJOR | frontend/src/components/PayerSelect.vue:17 | 已修正或核對，待重掃 | 已核對欄位 id 與標籤關係；自訂選擇器按鈕有可存取名稱。 |
| AaFt8ZiO7GRUxAPz9K4o | Web:InputWithoutLabelCheck | MAJOR | frontend/src/components/RoleSelect.vue:16 | 已修正或核對，待重掃 | 已核對欄位 id 與標籤關係；自訂選擇器按鈕有可存取名稱。 |
| AaFt8ZiN7GRUxAPz9K4n | Web:S6853 | MAJOR | frontend/src/components/CategoryManagement.vue:24 | 已修正或核對，待重掃 | 已修正標籤無法指向按鈕的自訂選擇器，改用可存取名稱。 |
| AaFt8ZiX7GRUxAPz9K48 | Web:S6853 | MAJOR | frontend/src/views/GroupsView.vue:45 | 已修正或核對，待重掃 | 已修正標籤無法指向按鈕的自訂選擇器，改用可存取名稱。 |
| AaFt8ZiX7GRUxAPz9K49 | Web:S6853 | MAJOR | frontend/src/views/GroupsView.vue:45 | 已修正或核對，待重掃 | 已修正標籤無法指向按鈕的自訂選擇器，改用可存取名稱。 |
| AaFt8ZiX7GRUxAPz9K4_ | typescript:S9011 | MAJOR | frontend/src/views/GroupsView.vue:45 | 已修正或核對，待重掃 | 核對目前 Vue template 的按鈕皆明確指定 button 或 submit。 |
| AaFt8Zib7GRUxAPz9K5G | typescript:S9011 | MAJOR | frontend/src/views/MembersView.vue:169 | 已修正或核對，待重掃 | 核對目前 Vue template 的按鈕皆明確指定 button 或 submit。 |
| AaFt8Zib7GRUxAPz9K5H | typescript:S9011 | MAJOR | frontend/src/views/MembersView.vue:177 | 已修正或核對，待重掃 | 核對目前 Vue template 的按鈕皆明確指定 button 或 submit。 |
| AaFt8Zil7GRUxAPz9K5b | typescript:S6671 | MAJOR | frontend/src/offline/db.ts:17 | 已修正或核對，待重掃 | IndexedDB Promise 以 request.error 或 Error 物件拒絕。 |
| AaFt8Zil7GRUxAPz9K5c | typescript:S6671 | MAJOR | frontend/src/offline/db.ts:27 | 已修正或核對，待重掃 | IndexedDB Promise 以 request.error 或 Error 物件拒絕。 |
| AaFt8ZiY7GRUxAPz9K5D | typescript:S9011 | MAJOR | frontend/src/views/SubscriptionsView.vue:227 | 已修正或核對，待重掃 | 核對目前 Vue template 的按鈕皆明確指定 button 或 submit。 |
| AaFt8Zif7GRUxAPz9K5P | Web:S6853 | MAJOR | frontend/src/views/ExpensesView.vue:117 | 已修正或核對，待重掃 | 已修正標籤無法指向按鈕的自訂選擇器，改用可存取名稱。 |
| AaFt8Zif7GRUxAPz9K5Q | Web:S6853 | MAJOR | frontend/src/views/ExpensesView.vue:117 | 已修正或核對，待重掃 | 已修正標籤無法指向按鈕的自訂選擇器，改用可存取名稱。 |
| AaFt8ZiY7GRUxAPz9K5B | Web:S6853 | MAJOR | frontend/src/views/SubscriptionsView.vue:224 | 已修正或核對，待重掃 | 已修正標籤無法指向按鈕的自訂選擇器，改用可存取名稱。 |
| AaFt8ZiY7GRUxAPz9K5C | Web:S6853 | MAJOR | frontend/src/views/SubscriptionsView.vue:224 | 已修正或核對，待重掃 | 已修正標籤無法指向按鈕的自訂選擇器，改用可存取名稱。 |
| AaFt8Zib7GRUxAPz9K5E | typescript:S9011 | MAJOR | frontend/src/views/MembersView.vue:127 | 已修正或核對，待重掃 | 核對目前 Vue template 的按鈕皆明確指定 button 或 submit。 |
| AaFt8Zis7GRUxAPz9K5r | docker:S6471 | MINOR | Dockerfile:27 | 已修正或核對，待重掃 | DHI runtime 預設 USER 65532:65532。 |
| AaFt8Zir7GRUxAPz9K5o | githubactions:S8233 | MAJOR | .github/workflows/release.yml:12 | 已修正或核對，待重掃 | write 權限由 workflow 層移到發布 job。 |
| AaFt8Zir7GRUxAPz9K5p | githubactions:S8233 | MAJOR | .github/workflows/release.yml:13 | 已修正或核對，待重掃 | write 權限由 workflow 層移到發布 job。 |
| AaFt8Zir7GRUxAPz9K5i | githubactions:S7637 | MAJOR | .github/workflows/release.yml:44 | 已修正或核對，待重掃 | workflow action 使用完整 commit SHA。 |
| AaFt8Zir7GRUxAPz9K5j | githubactions:S7637 | MAJOR | .github/workflows/release.yml:49 | 已修正或核對，待重掃 | workflow action 使用完整 commit SHA。 |
| AaFt8Zir7GRUxAPz9K5k | githubactions:S7637 | MAJOR | .github/workflows/release.yml:56 | 已修正或核對，待重掃 | workflow action 使用完整 commit SHA。 |
| AaFt8Zir7GRUxAPz9K5l | githubactions:S7637 | MAJOR | .github/workflows/release.yml:63 | 已修正或核對，待重掃 | workflow action 使用完整 commit SHA。 |
| AaFt8Zir7GRUxAPz9K5m | githubactions:S7637 | MAJOR | .github/workflows/release.yml:72 | 已修正或核對，待重掃 | workflow action 使用完整 commit SHA。 |
| AaFt8Zir7GRUxAPz9K5n | githubactions:S7637 | MAJOR | .github/workflows/release.yml:100 | 已修正或核對，待重掃 | workflow action 使用完整 commit SHA。 |
| AaFt8ZiS7GRUxAPz9K4q | Web:S6840 | MAJOR | frontend/src/views/AuthView.vue:93 | 已修正或核對，待重掃 | 已核對登入、註冊、密碼與 Email 欄位的 autocomplete 語意。 |
| AaFt8ZiS7GRUxAPz9K4r | Web:S6840 | MAJOR | frontend/src/views/AuthView.vue:97 | 已修正或核對，待重掃 | 已核對登入、註冊、密碼與 Email 欄位的 autocomplete 語意。 |
| AaFt8ZiS7GRUxAPz9K4s | typescript:S9011 | MAJOR | frontend/src/views/AuthView.vue:99 | 已修正或核對，待重掃 | 核對目前 Vue template 的按鈕皆明確指定 button 或 submit。 |
| AaFt8ZiP7GRUxAPz9K4p | Web:InputWithoutLabelCheck | MAJOR | frontend/src/components/BaseCombobox.vue:24 | 已修正或核對，待重掃 | 已核對欄位 id 與標籤關係；自訂選擇器按鈕有可存取名稱。 |
| AaFt8ZiM7GRUxAPz9K4m | Web:InputWithoutLabelCheck | MAJOR | frontend/src/components/CategoryIconPicker.vue:15 | 已修正或核對，待重掃 | 已核對欄位 id 與標籤關係；自訂選擇器按鈕有可存取名稱。 |
| AaFt8Zie7GRUxAPz9K5K | Web:S6853 | MAJOR | frontend/src/views/AdminView.vue:167 | 已修正或核對，待重掃 | 已修正標籤無法指向按鈕的自訂選擇器，改用可存取名稱。 |
| AaFt8Zie7GRUxAPz9K5L | Web:S6853 | MAJOR | frontend/src/views/AdminView.vue:168 | 已修正或核對，待重掃 | 已修正標籤無法指向按鈕的自訂選擇器，改用可存取名稱。 |
| AaFt8Zie7GRUxAPz9K5M | typescript:S9011 | MAJOR | frontend/src/views/AdminView.vue:195 | 已修正或核對，待重掃 | 核對目前 Vue template 的按鈕皆明確指定 button 或 submit。 |
| AaFt8Zie7GRUxAPz9K5N | typescript:S9011 | MAJOR | frontend/src/views/AdminView.vue:245 | 已修正或核對，待重掃 | 核對目前 Vue template 的按鈕皆明確指定 button 或 submit。 |
| AaFt8ZiW7GRUxAPz9K41 | Web:S6853 | MAJOR | frontend/src/views/ProfileView.vue:92 | 已修正或核對，待重掃 | 已修正標籤無法指向按鈕的自訂選擇器，改用可存取名稱。 |
| AaFt8ZiW7GRUxAPz9K45 | typescript:S9011 | MAJOR | frontend/src/views/ProfileView.vue:92 | 已修正或核對，待重掃 | 核對目前 Vue template 的按鈕皆明確指定 button 或 submit。 |
| AaFt8ZiH7GRUxAPz9K4g | Web:InputWithoutLabelCheck | MAJOR | frontend/src/components/CurrencySelect.vue:21 | 已修正或核對，待重掃 | 已核對欄位 id 與標籤關係；自訂選擇器按鈕有可存取名稱。 |
| AaFt8Zif7GRUxAPz9K5O | Web:S6853 | MAJOR | frontend/src/views/ExpensesView.vue:116 | 已修正或核對，待重掃 | 已修正標籤無法指向按鈕的自訂選擇器，改用可存取名稱。 |
| AaFt8Zif7GRUxAPz9K5R | typescript:S9011 | MAJOR | frontend/src/views/ExpensesView.vue:120 | 已修正或核對，待重掃 | 核對目前 Vue template 的按鈕皆明確指定 button 或 submit。 |
| AaFt8ZiY7GRUxAPz9K5A | Web:S6853 | MAJOR | frontend/src/views/SubscriptionsView.vue:223 | 已修正或核對，待重掃 | 已修正標籤無法指向按鈕的自訂選擇器，改用可存取名稱。 |
| AaFt8Zin7GRUxAPz9K5d | typescript:S7737 | MINOR | frontend/src/i18n.ts:25 | 已修正或核對，待重掃 | 移除函式參數的物件字面預設值。 |
| AaFt8ZiL7GRUxAPz9K4j | Web:S6853 | MAJOR | frontend/src/components/LanguageSwitcher.vue:34 | 已修正或核對，待重掃 | 已修正標籤無法指向按鈕的自訂選擇器，改用可存取名稱。 |
| AaFt8ZiJ7GRUxAPz9K4i | Web:InputWithoutLabelCheck | MAJOR | frontend/src/components/TimezoneSelect.vue:30 | 已修正或核對，待重掃 | 已核對欄位 id 與標籤關係；自訂選擇器按鈕有可存取名稱。 |
| AaFt8ZiW7GRUxAPz9K40 | Web:S6853 | MAJOR | frontend/src/views/ProfileView.vue:90 | 已修正或核對，待重掃 | 已修正標籤無法指向按鈕的自訂選擇器，改用可存取名稱。 |
| AaFt8Zis7GRUxAPz9K5q | docker:S6505 | MAJOR | Dockerfile:4 | 已修正或核對，待重掃 | Dockerfile 的 npm ci 使用 --ignore-scripts。 |
| AaFt8ZiB7GRUxAPz9K4f | go:S5332 | MINOR | backend/internal/mailer/mailer.go:49 | 已修正或核對，待重掃 | 移除獨立 SMTP 明文寄送實作，統一使用 PocketBase Native mailer。 |

Out of scope for this 77-issue export: six dependency risks and the analysis warning shown in the project summary require separate dependency/CI evidence. The follow-up security change makes SubFlow HTTPS-only; the public reverse proxy controls public TLS termination and path routing.
