local hs = {}

if obj.status == nil then
  hs.status = "Progressing"
  hs.message = "Waiting for Runner status"
  return hs
end

if obj.metadata.generation ~= nil and (obj.status.observedGeneration == nil or obj.status.observedGeneration < obj.metadata.generation) then
  hs.status = "Progressing"
  hs.message = "Waiting for Runner spec to be reconciled"
  return hs
end

local ready = nil
local reconciling = nil
if obj.status.conditions ~= nil then
  for _, condition in ipairs(obj.status.conditions) do
    if condition.type == "Stalled" and condition.status == "True" then
      hs.status = "Degraded"
      hs.message = condition.message
      return hs
    end
    if condition.type == "Ready" then
      ready = condition
    end
    if condition.type == "Reconciling" then
      reconciling = condition
    end
  end
end

if ready ~= nil and ready.status == "True" then
  hs.status = "Healthy"
  hs.message = ready.message
  return hs
end

hs.status = "Progressing"
if reconciling ~= nil and reconciling.message ~= nil then
  hs.message = reconciling.message
else
  hs.message = "Waiting for Runner to become ready"
end
return hs
