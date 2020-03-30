-- mycommand.lua
function lvar(varname, mode)
  if mode == "raw"
  then tex.sprint(varname)
  elseif mode == "num"
  then tex.sprint("\\num{" .. varname .. "}\\xspace")
  elseif mode == "percent"
  then tex.sprint("\\SI{" .. varname * 100 .. "}{\\percent}\\xspace")
  else assert(false, "invalid lvar mode: " .. mode)
  end
end

-- bool to number
btoi={ [true]=1, [false]=0 }

function ldiff(x, y, mode, units)
  relative = false
  absolute = false
  asPercent = false
  difference = false
  range = false
  low = false
  maximum = false
  minimum = false
  average = false
  fraction = false
  
  local function isempty(s)
     return s == nil or s == ''
  end

  if string.find(mode, "diff") then difference = true end
  if string.find(mode, "frac") then fraction = true end
  if string.find(mode, "range") then range = true end
  assert(btoi[difference] + btoi[range] + btoi[fraction] == 1,
         "format: you must specify either <difference> or <range> or <fraction>")

  if string.find(mode, "relative") then relative = true end
  if string.find(mode, "absolute") then absolute = true end
  
  if string.find(mode, "percent") then
     asPercent = true
     if isempty(units) then
        units = "\\percent"
     end
     if string.find(units, "none") then
        units = ""
     end
  end
  if string.find(mode, "low") then low = true end

  if string.find(mode, "min") then minimum = true end
  if string.find(mode, "max") then maximum = true end
  if string.find(mode, "avg") or string.find(mode, "average") then average = true end

  if fraction then
     assert(type(x) ~= "table", "tables not supported with <fraction> formatting")
     display = x / y
     
     if asPercent then tex.sprint("\\SI{" .. display * 100 .. "}{" .. units .. "}\\xspace")
     else              tex.sprint("\\SI{" .. display       .. "}{" .. units .. "}\\xspace")
     end
  elseif difference then
     assert(relative and not absolute or not relative and absolute, "format: you must specify either <relative> or <absolute>")

     if not (minimum or maximum or average) then
        if absolute and not low
        then nominal = x - y
        elseif absolute and low
        then nominal = y - x
        elseif relative and not low
        then nominal = x/y - 1.0
        elseif relative and low
        then nominal = 1.0 - x/y
        else assert(false)
        end
     elseif absolute then
        if low then
           a = y
           b = x
        else
           a = x
           b = y
        end
        if minimum then nominal = min_imp_abs(a,b)
        elseif maximum then nominal = max_imp_abs(a,b)
        elseif average then nominal = avg_imp_abs(a,b)
        else assert(false)
        end
     elseif relative then
        if     minimum and not low then nominal = min_imp_rel(x,y)
        elseif minimum and     low then nominal = min_imp_rel_low(x,y)
        elseif maximum and not low then nominal = max_imp_rel(x,y)
        elseif maximum and     low then nominal = max_imp_rel_low(x,y)
        elseif average and not low then nominal = avg_imp_rel(x,y)
        elseif average and     low then nominal = avg_imp_rel_low(x,y)
        else assert(false)
        end
     else assert(false)
     end

     display = nominal
     if asPercent then display = nominal * 100 end

     tex.sprint("\\SI{" .. display .. "}{" .. units .. "}\\xspace")
  elseif range then
     if type(x) == "table" then
        if     absolute and not low then
           lower   = min_imp_abs(x,y)
           greater = max_imp_abs(x,y)
        elseif absolute and     low then
           lower   = min_imp_abs(y,x)
           greater = max_imp_abs(y,x)
        elseif relative and not low then
           lower   = min_imp_rel(x,y)
           greater = max_imp_rel(x,y)
        elseif relative and low then
           lower   = min_imp_rel_low(x,y)
           greater = max_imp_rel_low(x,y)
        else assert(false)
        end
     else
        lower   = x
        greater = y
        if x > y then
           lower   = y
           greater = x
        end
     end

     if asPercent then tex.sprint("\\SIrange{" .. lower * 100 .. "}{" .. greater * 100 .. "}{" .. units .. "}\\xspace")
     else              tex.sprint("\\SIrange{" .. lower       .. "}{" .. greater       .. "}{" .. units .. "}\\xspace")
     end
  else assert(false)
  end
end

-- maximum of a lua table
-- warning: only works for numbers that are not smaller than -9999999
function tablemax (xs)
  local max = -9999999
  for k,v in pairs(xs) do
    if(max < v) then max = v end
  end
  return max
end

-- minimum of a lua table
-- warning: only works for numbers that are not greater than 9999999
function tablemin (xs)
  local min = 9999999
  for k,v in pairs(xs) do
    if(min > v) then min = v end
  end
  return min
end

-- average of a lua table
function tableavg (xs)
  sum = 0
  n = 0
  for k,v in pairs(xs) do
    sum = sum + v
    n = n + 1
  end
  return sum / n
end

-- returns the maximum improvement between table xs and table ys
function max_imp_abs (xs, ys)
  local diffs = {}
  for k,v in pairs(xs) do
    diffs[k] = v - ys[k]
  end
  return tablemax(diffs)
end

-- returns the minimum improvement between table xs and table ys
function min_imp_abs (xs, ys)
  local diffs = {}
  for k,v in pairs(xs) do
    diffs[k] = v - ys[k]
  end
  return tablemin(diffs)
end

-- returns the average improvement between table xs and table ys
function avg_imp_abs (xs, ys)
  local diffs = {}
  for k,v in pairs(xs) do
    diffs[k] = v - ys[k]
  end
  return tableavg(diffs)
end

-- returns the maximum improvement as a fraction between table xs and table ys
function max_imp_rel (xs, ys)
  local diffs = {}
  for k,v in pairs(xs) do
    diffs[k] = v / ys[k] - 1
  end
  return tablemax(diffs)
end

-- returns the minimum improvement as a fraction between table xs and table ys
function min_imp_rel (xs, ys)
  local diffs = {}
  for k,v in pairs(xs) do
    diffs[k] = v / ys[k] - 1
  end
  return tablemin(diffs)
end

-- returns the average improvement as a fraction between table xs and table ys
function avg_imp_rel (xs, ys)
  local diffs = {}
  for k,v in pairs(xs) do
    diffs[k] = v / ys[k] - 1
  end
  return tableavg(diffs)
end

-- returns the maximum improvement as a fraction between table xs and table ys where lower is better
function max_imp_rel_low (xs, ys)
  local diffs = {}
  for k,v in pairs(xs) do
    diffs[k] = 1 - v / ys[k]
  end
  return tablemax(diffs)
end

-- returns the minimum improvement as a fraction between table xs and table ys where lower is better
function min_imp_rel_low (xs, ys)
  local diffs = {}
  for k,v in pairs(xs) do
    diffs[k] = 1 - v / ys[k]
  end
  return tablemin(diffs)
end

-- returns the average improvement as a fraction between table xs and table ys where lower is better
function avg_imp_rel_low (xs, ys)
  local diffs = {}
  for k,v in pairs(xs) do
    diffs[k] = 1 - v / ys[k]
  end
  return tableavg(diffs)
end
