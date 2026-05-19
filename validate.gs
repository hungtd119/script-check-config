// validate.gs — Google Apps Script
// Cách dùng:
//   1. Mở Google Sheet, vào Extensions > Apps Script, paste toàn bộ file này vào.
//   2. Paste data từ activityopen.csv vào sheet đầu tiên (tên sheet tùy ý, hoặc đặt tên "Data").
//      Sheet phải giữ nguyên 4 dòng header đầu như file CSV gốc.
//   3. Reload Google Sheet → xuất hiện menu "Validate" ở thanh menu.
//   4. Click Validate > Chạy validation → kết quả ghi vào sheet "Results".

// ─── Menu ─────────────────────────────────────────────────────────────────────

function onOpen() {
  SpreadsheetApp.getUi()
    .createMenu('Validate')
    .addItem('Chạy validation', 'validate')
    .addSeparator()
    .addItem('[Debug] Rule 4/6 – xem dữ liệu thô', 'debugTimeOverlap')
    .addToUi();
}

// ─── Debug helper ─────────────────────────────────────────────────────────────

// Chạy function này để xem script đang đọc gì cho các activityId liên quan đến Rule 4 & 6.
// Kết quả ghi vào sheet "Debug" và Apps Script Logger (View > Logs).
function debugTimeOverlap() {
  const ss = SpreadsheetApp.getActiveSpreadsheet();
  const dataSheet = ss.getSheetByName('Data') || ss.getSheets()[0];
  const { rows } = parseSheet(dataSheet);

  const targetIds = new Set([2002, 2166, 2167, 2168, 7601, 7602, 7603, 7604]);
  const matched = rows.filter(r => targetIds.has(r.activityId));

  // Ghi vào sheet Debug
  let dbgSheet = ss.getSheetByName('Debug');
  if (dbgSheet) { dbgSheet.clearContents(); dbgSheet.clearFormats(); }
  else { dbgSheet = ss.insertSheet('Debug'); }

  const header = ['lineNum', 'id', 'activityId', 'startTimeType',
                  'startTime (raw)', 'endTime (raw)',
                  'parseTimeRange ok?', 'absStart', 'absEnd'];
  dbgSheet.getRange(1, 1, 1, header.length).setValues([header]).setFontWeight('bold');

  const out = matched.map(r => {
    const tr = parseTimeRange(r);
    return [
      r.lineNum,
      r.id,
      r.activityId,
      r.startTimeType,
      r.startTime,
      r.endTime,
      tr ? 'YES' : 'NO – bị skip',
      tr && tr.absolute ? tr.absStart.toISOString() : (tr ? `rel=${tr.relStart}` : ''),
      tr && tr.absolute ? tr.absEnd.toISOString()   : (tr ? `rel=${tr.relEnd}`   : ''),
    ];
  });

  if (out.length > 0) {
    dbgSheet.getRange(2, 1, out.length, header.length).setValues(out);
  } else {
    dbgSheet.getRange(2, 1).setValue('Không tìm thấy dòng nào với activityId 2002/2166/2167/2168/7601-7604');
  }

  // Cũng log ra console
  Logger.log(`Tìm thấy ${matched.length} dòng với activityId thuộc Rule 4/6`);
  matched.forEach(r => {
    const tr = parseTimeRange(r);
    Logger.log(`Line ${r.lineNum}: actId=${r.activityId} startTimeType=${r.startTimeType} startTime="${r.startTime}" endTime="${r.endTime}" → parseTimeRange=${tr ? 'OK' : 'NULL'}`);
  });

  ss.setActiveSheet(dbgSheet);
  SpreadsheetApp.getUi().alert(`Tìm thấy ${matched.length} dòng.\nXem sheet "Debug" và View > Logs để biết chi tiết.`);
}

// ─── Main entry ───────────────────────────────────────────────────────────────

function validate() {
  const ss = SpreadsheetApp.getActiveSpreadsheet();
  const dataSheet = ss.getSheetByName('Data') || ss.getSheets()[0];

  const { rows, warnings } = parseSheet(dataSheet);
  const results = runAllRules(rows);
  writeReport(ss, results, warnings);
}

// ─── Parse sheet ──────────────────────────────────────────────────────────────

// Khi user paste CSV vào Sheets, các ô datetime bị auto-convert thành Date object.
// getDisplayValues() trả về chuỗi theo locale (VD: "1/26/2026 5:00:00 AM") → parseDateTime fail.
// Fix: dùng getValues() (trả về Date object) cho dữ liệu thực,
//      getDisplayValues() chỉ dùng cho rawFields (Rule 1 check khoảng trắng).

function normalizeTimeField(val) {
  if (!val && val !== 0) return '';
  if (val instanceof Date) {
    const p = n => String(n).padStart(2, '0');
    return `${val.getFullYear()}/${p(val.getMonth()+1)}/${p(val.getDate())} ${p(val.getHours())}:${p(val.getMinutes())}:${p(val.getSeconds())}`;
  }
  return String(val).trim();
}

function parseSheet(sheet) {
  const displayVals = sheet.getDataRange().getDisplayValues(); // cho rawFields (Rule 1)
  const rawVals     = sheet.getDataRange().getValues();        // giá trị thực (Date, number)
  const rows = [];
  const warnings = [];

  // 4 dòng đầu là header metadata — bỏ qua
  for (let i = 4; i < displayVals.length; i++) {
    const dRec = displayVals[i]; // display strings
    const vRec = rawVals[i];     // actual values
    const lineNum = i + 1;

    const idStr = dRec[0].trim();
    if (idStr === '') continue; // dòng trống hoặc label cụm

    // Lấy số nguyên từ rawVals: Sheets trả về number (float), dùng Math.round để chắc chắn
    const getInt = (idx, name) => {
      const v = vRec[idx];
      if (v === '' || v === undefined || v === null) return 0;
      if (typeof v === 'number') return Math.round(v);
      const n = parseInt(String(v).trim(), 10);
      if (isNaN(n)) {
        warnings.push(`Line ${lineNum}: không parse được ${name}="${v}" thành int`);
        return 0;
      }
      return n;
    };
    // Lấy chuỗi từ rawVals (JSON, text fields)
    const getStr = (idx) => {
      const v = vRec[idx];
      return (v === undefined || v === null) ? '' : String(v).trim();
    };
    // Lấy datetime field: nếu Sheets đã convert → Date object → chuẩn hóa thành YYYY/MM/DD HH:MM:SS
    const getTime = (idx) => normalizeTimeField(vRec[idx]);

    rows.push({
      lineNum,
      rawFields: dRec, // display strings cho Rule 1
      id:              getInt(0,  'id'),
      server:          getStr(1),
      exceptServer:    getStr(2),
      activityId:      getInt(3,  'activityId'),
      activityGroup:   getInt(4,  'activityGroup'),
      activityExclude: getStr(5),
      note:            getStr(6),
      isOpen:          getInt(7,  'isopen'),
      area:            getInt(8,  'area'),
      acType:          getInt(9,  'acType'),
      seType:          getInt(10, 'seType'),
      startTimeType:   getInt(11, 'startTimeType'),
      startTime:       getTime(12),
      endTime:         getTime(13),
      appearTime:      getTime(14),
      disappearTime:   getTime(15),
      effectiveTime:   getTime(16),
      levelVis:        getInt(17, 'levelVis'),
      vipVis:          getInt(18, 'vipVis'),
      conditionVis:    getInt(19, 'conditionVis'),
      conditionNum:    getInt(20, 'conditionNum'),
      serverLimit:     getInt(21, 'serverLimit'),
      playerLimit:     getInt(22, 'playerLimit'),
      reissue:         getInt(23, 'reissue'),
      plate:           getInt(24, 'plate'),
      channelType:     getInt(25, 'channelType'),
      channel:         getInt(26, 'channel'),
      integral:        getInt(27, 'integral'),
    });
  }

  return { rows, warnings };
}

// ─── Datetime helpers ─────────────────────────────────────────────────────────

// Format thực tế trong file có 2 biến thể cần xử lý:
//   "2026/02/20 5/00/00"   – giờ 1 chữ số, dấu / trong phần time
//   "2026/03/12  5/00/00"  – 2 khoảng trắng giữa date và time
function parseDateTime(s) {
  if (s instanceof Date) return s;
  s = (s || '').trim();
  if (!s) return null;

  // Dùng /\s+/ để xử lý 1 hoặc nhiều khoảng trắng giữa date và time
  const parts = s.split(/\s+/);
  if (parts.length < 2) return null;

  const dateSeg = parts[0].replace(/\//g, '-');               // YYYY/MM/DD → YYYY-MM-DD
  const rawTime = parts[parts.length - 1].replace(/\//g, ':'); // H/MM/SS → H:MM:SS

  // Pad giờ 1 chữ số: "5:00:00" → "05:00:00" (ISO 8601 yêu cầu 2 chữ số)
  const timeSeg = rawTime.replace(/^(\d):/, '0$1:');

  const d = new Date(`${dateSeg}T${timeSeg}`);
  return isNaN(d.getTime()) ? null : d;
}

// ─── Server range validation (Rule 3) ────────────────────────────────────────

function validateServerRange(s) {
  s = s.trim();
  if (!s) return null;

  let outer;
  try {
    outer = JSON.parse(s);
  } catch (e) {
    return `không parse được JSON: ${e.message}`;
  }

  if (!Array.isArray(outer) || outer.length === 0) return 'mảng rỗng hoặc không hợp lệ';

  for (const elem of outer) {
    if (Array.isArray(elem)) {
      if (elem.length === 0) return 'inner array rỗng';
      for (const v of elem) {
        if (typeof v !== 'number' || v < 0 || v > 9999999999)
          return `giá trị ${v} nằm ngoài [0, 9999999999]`;
      }
      if (elem.length === 2 && elem[0] >= elem[1])
        return `start=${elem[0]} >= end=${elem[1]}`;
    } else {
      if (typeof elem !== 'number' || elem < 0 || elem > 9999999999)
        return `giá trị ${elem} nằm ngoài [0, 9999999999]`;
    }
  }
  return null;
}

// ─── Time range helpers (Rules 4/6/8/12) ─────────────────────────────────────

function parseTimeRange(row) {
  if (row.startTimeType === 1) {
    const s = parseDateTime(row.startTime);
    const e = parseDateTime(row.endTime);
    if (!s || !e) return null;
    return { absolute: true, absStart: s, absEnd: e };
  }
  if (row.startTimeType === 2 || row.startTimeType === 3) {
    const s = parseFloat(row.startTime);
    const e = parseFloat(row.endTime);
    if (isNaN(s) || isNaN(e)) return null;
    return { absolute: false, relStart: s, relEnd: e };
  }
  return null;
}

// Kiểm tra [b.start, b.end] có giao với [a.start−buf, a.end+buf] không
function overlapsWithBuffer(trA, trB, bufDays) {
  if (trA.absolute !== trB.absolute) return false;
  if (trA.absolute) {
    const buf = bufDays * 24 * 60 * 60 * 1000;
    return trB.absStart < new Date(trA.absEnd.getTime() + buf) &&
           trB.absEnd   > new Date(trA.absStart.getTime() - buf);
  }
  const buf = bufDays * 24;
  return trB.relStart < trA.relEnd + buf && trB.relEnd > trA.relStart - buf;
}

function overlaps(trA, trB) {
  return overlapsWithBuffer(trA, trB, 0);
}

// ─── Shared rule helpers ──────────────────────────────────────────────────────

function checkTimeOverlap(rows, activityIDs, ruleNum, name) {
  const errors = [];
  const grouped = {};

  for (const row of rows) {
    if (activityIDs.has(row.activityId)) {
      if (!grouped[row.activityId]) grouped[row.activityId] = [];
      grouped[row.activityId].push(row);
    }
  }

  for (const group of Object.values(grouped)) {
    for (let i = 0; i < group.length; i++) {
      for (let j = i + 1; j < group.length; j++) {
        const a = group[i], b = group[j];
        const trA = parseTimeRange(a), trB = parseTimeRange(b);
        if (!trA || !trB) continue;
        if (overlapsWithBuffer(trA, trB, 7)) {
          errors.push({
            lineNum: b.lineNum, rowId: b.id,
            message: `activityId=${b.activityId} chồng thời gian (±7 ngày) với line ${a.lineNum} (id=${a.id})`,
          });
        }
      }
    }
  }

  return { ruleNum, name, errors };
}

function checkRequiredFields(rows, activityIDs, reqs, ruleNum, name) {
  const errors = [];
  for (const row of rows) {
    if (!activityIDs.has(row.activityId)) continue;
    for (const { field, expected, get } of reqs) {
      const got = get(row);
      if (got !== expected) {
        errors.push({
          lineNum: row.lineNum, rowId: row.id,
          message: `activityId=${row.activityId}  ${field}=${got} (expected ${expected})`,
        });
      }
    }
  }
  return { ruleNum, name, errors };
}

// ─── 14 Rules ─────────────────────────────────────────────────────────────────

function runAllRules(rows) {
  return [
    rule1(rows),  rule2(rows),  rule3(rows),
    rule4(rows),  rule5(rows),  rule6(rows),
    rule7(rows),  rule8(rows),  rule9(rows),
    rule10(rows), rule11(rows), rule12(rows),
    rule13(rows), rule14(rows),
  ];
}

// Rule 1 – Ô trống có dấu cách
function rule1(rows) {
  const errors = [];
  for (const row of rows) {
    for (let i = 0; i < row.rawFields.length; i++) {
      const val = String(row.rawFields[i]);
      if (val === '') continue;
      if (val.trim() === '') {
        errors.push({
          lineNum: row.lineNum, rowId: row.id,
          message: `cột ${i}: ô chứa khoảng trắng (phải là "")`,
        });
      }
    }
  }
  return { ruleNum: 1, name: 'Ô trống có dấu cách', errors };
}

// Rule 2 – Trùng ID
function rule2(rows) {
  const errors = [];
  const seen = {};
  for (const row of rows) {
    if (seen[row.id] !== undefined) {
      errors.push({
        lineNum: row.lineNum, rowId: row.id,
        message: `id=${row.id} đã xuất hiện ở line ${seen[row.id]}`,
      });
    } else {
      seen[row.id] = row.lineNum;
    }
  }
  return { ruleNum: 2, name: 'Trùng ID', errors };
}

// Rule 3 – Khoảng serverID không chính xác
function rule3(rows) {
  const errors = [];
  for (const row of rows) {
    if (row.server) {
      const err = validateServerRange(row.server);
      if (err) errors.push({ lineNum: row.lineNum, rowId: row.id, message: `server="${row.server}" → ${err}` });
    }
    if (row.exceptServer) {
      const err = validateServerRange(row.exceptServer);
      if (err) errors.push({ lineNum: row.lineNum, rowId: row.id, message: `except_server="${row.exceptServer}" → ${err}` });
    }
  }
  return { ruleNum: 3, name: 'Khoảng serverID không chính xác', errors };
}

// Rule 4 – Binh Pháp Nghiên Tập: chồng thời gian
function rule4(rows) {
  return checkTimeOverlap(rows, new Set([2002, 2166, 2167, 2168]), 4, 'Binh Pháp Nghiên Tập – chồng thời gian (±7 ngày)');
}

// Rule 5 – Binh Pháp Nghiên Tập: yêu cầu field
function rule5(rows) {
  return checkRequiredFields(rows, new Set([2002, 2166, 2167, 2168]), [
    { field: 'isopen',      expected: 1,   get: r => r.isOpen },
    { field: 'area',        expected: 2,   get: r => r.area },
    { field: 'acType',      expected: 159, get: r => r.acType },
    { field: 'seType',      expected: 159, get: r => r.seType },
    { field: 'levelVis',    expected: 55,  get: r => r.levelVis },
    { field: 'vipVis',      expected: 0,   get: r => r.vipVis },
    { field: 'serverLimit', expected: 168, get: r => r.serverLimit },
    { field: 'reissue',     expected: 2,   get: r => r.reissue },
  ], 5, 'Binh Pháp Nghiên Tập – yêu cầu field');
}

// Rule 6 – Tướng UR: chồng thời gian
function rule6(rows) {
  return checkTimeOverlap(rows, new Set([7601, 7602, 7603, 7604]), 6, 'Tướng UR – chồng thời gian (±7 ngày)');
}

// Rule 7 – Tướng UR: yêu cầu field
function rule7(rows) {
  return checkRequiredFields(rows, new Set([7601, 7602, 7603, 7604]), [
    { field: 'isopen',      expected: 1,   get: r => r.isOpen },
    { field: 'area',        expected: 1,   get: r => r.area },
    { field: 'acType',      expected: 165, get: r => r.acType },
    { field: 'seType',      expected: 165, get: r => r.seType },
    { field: 'levelVis',    expected: 10,  get: r => r.levelVis },
    { field: 'vipVis',      expected: 0,   get: r => r.vipVis },
    { field: 'serverLimit', expected: 168, get: r => r.serverLimit },
    { field: 'reissue',     expected: 1,   get: r => r.reissue },
    { field: 'integral',    expected: 1,   get: r => r.integral },
  ], 7, 'Tướng UR – yêu cầu field');
}

// Rule 8 – Vũ Khí Chuyên Dụng: chồng thời gian
function rule8(rows) {
  return checkTimeOverlap(rows, new Set([80102, 80103, 80104, 80105, 80106]), 8, 'Vũ Khí Chuyên Dụng – chồng thời gian (±7 ngày)');
}

// Rule 9 – Vũ Khí Chuyên Dụng: yêu cầu field
function rule9(rows) {
  return checkRequiredFields(rows, new Set([80102, 80103, 80104, 80105, 80106]), [
    { field: 'isopen',      expected: 1,   get: r => r.isOpen },
    { field: 'area',        expected: 1,   get: r => r.area },
    { field: 'acType',      expected: 167, get: r => r.acType },
    { field: 'seType',      expected: 167, get: r => r.seType },
    { field: 'levelVis',    expected: 26,  get: r => r.levelVis },
    { field: 'vipVis',      expected: 0,   get: r => r.vipVis },
    { field: 'serverLimit', expected: 168, get: r => r.serverLimit },
    { field: 'reissue',     expected: 2,   get: r => r.reissue },
  ], 9, 'Vũ Khí Chuyên Dụng – yêu cầu field');
}

// Rule 10 – serverLimit phải là 0 hoặc 168 khi startTimeType=1
function rule10(rows) {
  const errors = [];
  for (const row of rows) {
    if (row.startTimeType !== 1) continue;
    if (row.serverLimit !== 0 && row.serverLimit !== 168) {
      errors.push({
        lineNum: row.lineNum, rowId: row.id,
        message: `startTimeType=1  serverLimit=${row.serverLimit} (chỉ cho phép 0 hoặc 168)`,
      });
    }
  }
  return { ruleNum: 10, name: 'serverLimit khi startTimeType=1', errors };
}

// Rule 11 – Format datetime khi startTimeType=1
// Check bằng parseDateTime (giống Go script): nếu không parse được → lỗi.
function rule11(rows) {
  const errors = [];
  const timeFields = ['startTime', 'endTime', 'appearTime', 'disappearTime'];

  for (const row of rows) {
    if (row.startTimeType !== 1) continue;
    for (const field of timeFields) {
      const val = row[field];
      if (!val) continue;
      if (!parseDateTime(val)) {
        errors.push({
          lineNum: row.lineNum, rowId: row.id,
          message: `${field}="${val}" sai format (không parse được datetime)`,
        });
      }
    }
  }
  return { ruleNum: 11, name: 'Format datetime khi startTimeType=1', errors };
}

// Rule 12 – Sự kiện song hành phải cùng thời gian
function rule12(rows) {
  const errors = [];
  const groups = [
    { setA: new Set([3510]),           setB: new Set([3511]), label: '3510↔3511' },
    { setA: new Set([194, 355, 356]),  setB: new Set([156]),  label: '194/355/356↔156' },
    { setA: new Set([7002]),           setB: new Set([299]),  label: '7002↔299' },
  ];

  for (const g of groups) {
    const setARows = rows.filter(r => g.setA.has(r.activityId));
    const setBRows = rows.filter(r => g.setB.has(r.activityId));

    for (const a of setARows) {
      for (const b of setBRows) {
        const trA = parseTimeRange(a), trB = parseTimeRange(b);
        if (!trA || !trB || !overlaps(trA, trB)) continue;

        const check = (fieldName, valA, valB) => {
          if (valA !== valB) {
            errors.push({
              lineNum: b.lineNum, rowId: b.id,
              message: `[${g.label}] ${fieldName} không khớp: line ${a.lineNum} (actId=${a.activityId}) "${valA}" ≠ line ${b.lineNum} (actId=${b.activityId}) "${valB}"`,
            });
          }
        };
        check('startTime',    a.startTime,    b.startTime);
        check('appearTime',   a.appearTime,   b.appearTime);
        check('endTime',      a.endTime,      b.endTime);
        check('disappearTime',a.disappearTime,b.disappearTime);
      }
    }
  }
  return { ruleNum: 12, name: 'Sự kiện song hành phải cùng thời gian', errors };
}

// Rule 13 – startTime=appearTime, endTime=disappearTime (tất cả dòng)
function rule13(rows) {
  const errors = [];
  for (const row of rows) {
    if (row.startTime !== row.appearTime) {
      errors.push({
        lineNum: row.lineNum, rowId: row.id,
        message: `startTime="${row.startTime}" ≠ appearTime="${row.appearTime}"`,
      });
    }
    if (row.endTime !== row.disappearTime) {
      errors.push({
        lineNum: row.lineNum, rowId: row.id,
        message: `endTime="${row.endTime}" ≠ disappearTime="${row.disappearTime}"`,
      });
    }
  }
  return { ruleNum: 13, name: 'startTime=appearTime & endTime=disappearTime', errors };
}

// Rule 14 – Quỹ tháng cố định (activityId 5013–5025)
function rule14(rows) {
  const errors = [];
  const schedule = {
    5013: { startMonth: 1,  endMonth: 2  },
    5014: { startMonth: 2,  endMonth: 3  },
    5015: { startMonth: 3,  endMonth: 4  },
    5016: { startMonth: 4,  endMonth: 5  },
    5017: { startMonth: 5,  endMonth: 6  },
    5019: { startMonth: 6,  endMonth: 7  },
    5020: { startMonth: 7,  endMonth: 8  },
    5021: { startMonth: 8,  endMonth: 9  },
    5022: { startMonth: 9,  endMonth: 10 },
    5023: { startMonth: 10, endMonth: 11 },
    5024: { startMonth: 11, endMonth: 12 },
    5025: { startMonth: 12, endMonth: 1, nextYear: true },
  };

  const fieldReqs = [
    { field: 'isopen',        expected: 1,   get: r => r.isOpen },
    { field: 'area',          expected: 1,   get: r => r.area },
    { field: 'acType',        expected: 129, get: r => r.acType },
    { field: 'seType',        expected: 129, get: r => r.seType },
    { field: 'startTimeType', expected: 5,   get: r => r.startTimeType },
    { field: 'levelVis',      expected: 26,  get: r => r.levelVis },
    { field: 'vipVis',        expected: 0,   get: r => r.vipVis },
    { field: 'serverLimit',   expected: 168, get: r => r.serverLimit },
    { field: 'playerLimit',   expected: 0,   get: r => r.playerLimit },
    { field: 'reissue',       expected: 1,   get: r => r.reissue },
    { field: 'plate',         expected: 0,   get: r => r.plate },
  ];

  for (const row of rows) {
    const spec = schedule[row.activityId];
    if (!spec) continue;

    for (const { field, expected, get } of fieldReqs) {
      const got = get(row);
      if (got !== expected) {
        errors.push({
          lineNum: row.lineNum, rowId: row.id,
          message: `activityId=${row.activityId}  ${field}=${got} (expected ${expected})`,
        });
      }
    }

    const st = parseDateTime(row.startTime);
    if (st) {
      if (st.getMonth() + 1 !== spec.startMonth || st.getDate() !== 1 ||
          st.getHours() !== 5 || st.getMinutes() !== 0 || st.getSeconds() !== 0) {
        errors.push({
          lineNum: row.lineNum, rowId: row.id,
          message: `activityId=${row.activityId}  startTime="${row.startTime}": expected tháng ${String(spec.startMonth).padStart(2,'0')} ngày 01 lúc 05:00:00`,
        });
      }
    } else {
      errors.push({
        lineNum: row.lineNum, rowId: row.id,
        message: `activityId=${row.activityId}  startTime="${row.startTime}" không parse được datetime`,
      });
    }

    const et = parseDateTime(row.endTime);
    if (et) {
      if (et.getMonth() + 1 !== spec.endMonth || et.getDate() !== 1 ||
          et.getHours() !== 5 || et.getMinutes() !== 0 || et.getSeconds() !== 0) {
        errors.push({
          lineNum: row.lineNum, rowId: row.id,
          message: `activityId=${row.activityId}  endTime="${row.endTime}": expected tháng ${String(spec.endMonth).padStart(2,'0')} ngày 01 lúc 05:00:00`,
        });
      }
      if (spec.nextYear && st && et.getFullYear() !== st.getFullYear() + 1) {
        errors.push({
          lineNum: row.lineNum, rowId: row.id,
          message: `activityId=5025  endTime năm ${et.getFullYear()} phải = startTime năm ${st.getFullYear()} + 1`,
        });
      }
    } else {
      errors.push({
        lineNum: row.lineNum, rowId: row.id,
        message: `activityId=${row.activityId}  endTime="${row.endTime}" không parse được datetime`,
      });
    }
  }
  return { ruleNum: 14, name: 'Quỹ tháng cố định (5013–5025)', errors };
}

// ─── Write report to "Results" sheet ─────────────────────────────────────────

const CLR_HEADER  = '#1e4d78';
const CLR_PASS    = '#d9ead3';
const CLR_FAIL    = '#f4cccc';
const CLR_ERR_TXT = '#cc0000';
const CLR_WARN    = '#e65100';
const CLR_SUMMARY_PASS = '#34a853';
const CLR_SUMMARY_FAIL = '#ea4335';

function writeReport(ss, results, warnings) {
  let sheet = ss.getSheetByName('Results');
  if (sheet) {
    sheet.clearContents();
    sheet.clearFormats();
  } else {
    sheet = ss.insertSheet('Results');
  }

  let row = 1;

  // Title
  setMergedRow(sheet, row, 1, 4, 'VALIDATION REPORT', {
    bg: CLR_HEADER, color: '#ffffff', bold: true, size: 14,
  });
  row++;

  setMergedRow(sheet, row, 1, 4, `Chạy lúc: ${new Date().toLocaleString('vi-VN')}`);
  row++;

  // Warnings
  if (warnings.length > 0) {
    row++;
    sheet.getRange(row, 1).setValue('⚠ Cảnh báo parse:').setFontWeight('bold');
    row++;
    for (const w of warnings) {
      setMergedRow(sheet, row, 1, 4, w, { color: CLR_WARN });
      row++;
    }
  }
  row++;

  // Column headers
  const hdr = sheet.getRange(row, 1, 1, 4);
  hdr.setValues([['Rule', 'Tên Rule', 'Trạng thái', 'Số lỗi']]);
  hdr.setBackground('#cccccc').setFontWeight('bold');
  row++;

  let passCount = 0, failCount = 0;

  for (const r of results) {
    const pass = r.errors.length === 0;
    pass ? passCount++ : failCount++;

    const ruleRange = sheet.getRange(row, 1, 1, 4);
    ruleRange.setValues([[r.ruleNum, r.name, pass ? '✅ PASS' : '❌ FAIL', pass ? '' : r.errors.length]]);
    ruleRange.setBackground(pass ? CLR_PASS : CLR_FAIL);
    if (!pass) ruleRange.setFontWeight('bold');
    row++;

    if (!pass) {
      for (const e of r.errors) {
        const errRange = sheet.getRange(row, 1, 1, 4);
        errRange.setValues([[`   └─ Line ${e.lineNum}`, `id=${e.rowId}`, e.message, '']]);
        errRange.setFontColor(CLR_ERR_TXT);
        row++;
      }
    }
  }

  row++;

  // Summary footer
  const sumText = `Kết quả: ${passCount}/${results.length} PASS  |  ${failCount}/${results.length} FAIL`;
  setMergedRow(sheet, row, 1, 4, sumText, {
    bg: failCount === 0 ? CLR_SUMMARY_PASS : CLR_SUMMARY_FAIL,
    color: '#ffffff',
    bold: true,
    size: 12,
  });

  sheet.setColumnWidth(2, 320);
  sheet.setColumnWidth(3, 400);
  sheet.autoResizeColumns(1, 1);
  sheet.autoResizeColumns(4, 1);

  ss.setActiveSheet(sheet);
  SpreadsheetApp.getUi().alert(
    `Validation hoàn tất!\n\n${passCount}/${results.length} PASS   ${failCount}/${results.length} FAIL\n\nXem chi tiết ở sheet "Results".`
  );
}

function setMergedRow(sheet, row, startCol, colSpan, value, opts = {}) {
  const range = sheet.getRange(row, startCol, 1, colSpan);
  if (colSpan > 1) range.merge();
  range.setValue(value);
  if (opts.bg)    range.setBackground(opts.bg);
  if (opts.color) range.setFontColor(opts.color);
  if (opts.bold)  range.setFontWeight('bold');
  if (opts.size)  range.setFontSize(opts.size);
}
