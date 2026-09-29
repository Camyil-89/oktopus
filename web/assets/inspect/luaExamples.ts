/**
 * Примеры Lua для правил инспекции исходящих HTTP-запросов (MITM / http://).
 * Каждый скрипт обязан определять function inspect(ctx).
 * Возврат false / "allow" — правило не сработало; true / "match" — сработало (действие из настроек правила).
 * Порядок: при совпадении «Запретить» — сразу блок; «Разрешить» — проверяем следующие правила.
 */

export type InspectLuaExample = {
  id: string;
  title: string;
  description: string;
  script: string;
};

export const INSPECT_LUA_EXAMPLES: InspectLuaExample[] = [
  {
    id: "any-file-upload-heuristic",
    title: "Похоже на отправку файла (эвристика по заголовкам)",
    description:
      "POST/PUT/PATCH: multipart, бинарные и медиа Content-Type, Content-Disposition, имена файлов из тела (лимит 4 МБ). Пустой Content-Type без признаков файла не блокируется.",
    script: `-- «Запретить» — сразу блок; «Разрешить» — дальше по списку правил.
local PLAIN_TYPES = {
  "application/octet-stream",
  "application/pdf",
  "application/zip",
  "application/x-zip-compressed",
  "application/gzip",
  "application/x-gzip",
  "application/x-tar",
  "application/x-7z-compressed",
  "application/vnd.rar",
  "application/x-rar-compressed",
}

local function filenames_from_ctx(ctx)
  local out = {}
  local n = ctx.upload_filenames
  if n == nil then
    return out
  end
  for i = 1, #n do
    local name = n[i]
    if name ~= nil and name ~= "" then
      out[#out + 1] = name
    end
  end
  return out
end

function inspect(ctx)
  local method = ctx.method or ""
  if method ~= "POST" and method ~= "PUT" and method ~= "PATCH" then
    return false
  end

  local files = filenames_from_ctx(ctx)
  ctx:log("checked_mime_types", PLAIN_TYPES)
  ctx:log("upload_filenames", files)

  local cd = ctx:header("Content-Disposition")
  if cd ~= "" and string.lower(cd):find("filename=", 1, true) then
    ctx:log("match_reason", "content_disposition")
    return true
  end
  if #files > 0 then
    ctx:log("match_reason", "upload_filenames")
    return true
  end

  local ct = string.lower(ctx.content_type or "")
  if ct == "" then
    return false
  end

  if ct:find("multipart/", 1, true) then
    ctx:log("match_reason", "multipart")
    return true
  end

  for i = 1, #PLAIN_TYPES do
    if ct:find(PLAIN_TYPES[i], 1, true) then
      ctx:log("match_reason", "content_type")
      ctx:log("matched_mime", PLAIN_TYPES[i])
      return true
    end
  end

  if ct:find("^image/", 1) or ct:find("^video/", 1) or ct:find("^audio/", 1) then
    ctx:log("match_reason", "media_prefix")
    return true
  end

  if ct:find("^application/vnd%.", 1) or ct:find("^application/msword", 1) then
    ctx:log("match_reason", "office_legacy")
    return true
  end

  if ct:find("application/vnd%.openxmlformats%-officedocument", 1, true) then
    ctx:log("match_reason", "office_openxml")
    return true
  end

  return false
end`,
  },
  {
    id: "post-multipart",
    title: "POST с multipart (быстро)",
    description:
      "Только метод и Content-Type, без разбора тела. Подходит для грубой блокировки форм upload.",
    script: `function inspect(ctx)
  if ctx.method ~= "POST" and ctx.method ~= "PUT" then
    return false
  end
  local ct = ctx.content_type
  if ct == nil or ct == "" then
    return false
  end
  -- plain search (4-й аргумент true) быстрее regex
  if ct:find("multipart/form-data", 1, true) then
    return true
  end
  return false
end`,
  },
  {
    id: "host-upload-path",
    title: "Путь upload на хосте",
    description: "SNI/Host + PATH. Ранний выход, если хост не тот.",
    script: `function inspect(ctx)
  local host = ctx.host or ""
  if not host:find("dropbox%.com", 1) then
    return false
  end
  local path = ctx.path or "/"
  if path:find("/upload", 1, true) then
    return true
  end
  return false
end`,
  },
  {
    id: "large-body",
    title: "Большой исходящий body",
    description:
      "Content-Length из заголовка (−1 если chunked). Не читает тело.",
    script: `local LIMIT = 50 * 1024 * 1024

function inspect(ctx)
  if ctx.method ~= "POST" and ctx.method ~= "PUT" then
    return false
  end
  local n = ctx.content_length
  if n == nil or n < 0 then
    return false
  end
  if n >= LIMIT then
    return true
  end
  return false
end`,
  },
  {
    id: "group-external",
    title: "Группа + внешний хост",
    description: "ctx.has_group и простая проверка домена.",
    script: `function inspect(ctx)
  if not ctx:has_group("intern") then
    return false
  end
  local host = (ctx.host or ""):lower()
  if host == "" then
    return false
  end
  if host:find("%.corp%.local$") then
    return false
  end
  if ctx.method == "POST" or ctx.method == "PUT" then
    return true
  end
  return false
end`,
  },
  {
    id: "header-api",
    title: "Заголовок X-Requested-With",
    description: "ctx.header(name) — один заголовок без копирования всех.",
    script: `function inspect(ctx)
  if ctx.method ~= "POST" then
    return false
  end
  local xrw = ctx:header("X-Requested-With")
  if xrw == "XMLHttpRequest" then
    local ct = ctx.content_type or ""
    if ct:find("application/json", 1, true) then
      return false
    end
    if ct:find("multipart/", 1, true) then
      return true
    end
  end
  return false
end`,
  },
  {
    id: "whitelist-octet-stream",
    title: "Разрешить только octet-stream для пользователя",
    description:
      "При совпадении выберите в правиле «Разрешить» — запрос пройдёт (раньше других deny-правил не должно быть).",
    script: `function inspect(ctx)
  if ctx.user == nil or ctx.user == "" then
    return false
  end
  local ct = ctx.content_type or ""
  if ct:find("application/octet-stream", 1, true) then
    return true
  end
  return false
end`,
  },
];

export const INSPECT_LUA_API_REFERENCE = `Поля ctx (исходящий запрос):
  ctx.method          — GET, POST, …
  ctx.host            — SNI / Host
  ctx.path            — путь URL
  ctx.url             — полный URL
  ctx.content_type    — Content-Type
  ctx.content_length  — число (−1 если неизвестно)
  ctx.user            — имя proxy-auth
  ctx.client_ip       — адрес клиента
  ctx.groups          — таблица групп (1..n)
  ctx:header("Name")  — заголовок
  ctx:has_group("x")  — bool
  ctx.upload_filenames — таблица имён файлов (из заголовков и multipart-тела, лимит 4 МБ)
  ctx:log("key", value) — записать в extra журнала под id этого правила
  ctx:log({ key = value, ... }) — несколько полей сразу

Возврат inspect:
  false / nil / "allow" — правило не подошло
  true / "match"        — подошло (в правиле: запретить или разрешить)

Порядок правил: «Запретить» при совпадении останавливает проверку; «Разрешить» — только отмечает совпадение и идёт дальше. Whitelist: несколько «Разрешить» сверху, внизу жёсткие «Запретить».

Работает только когда прокси видит HTTP (режим MITM или http://). В tunnel HTTPS внутри туннеля не анализируется.`;
