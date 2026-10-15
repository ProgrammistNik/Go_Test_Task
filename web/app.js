const categories = {
  "": "Все категории",
  food: "Еда",
  transport: "Транспорт",
  housing: "Жильё",
  entertainment: "Развлечения",
  health: "Здоровье",
  shopping: "Покупки",
  other: "Другое"
}

const elements = {
  list: document.querySelector("#expense-list"),
  empty: document.querySelector("#empty"),
  status: document.querySelector("#status"),
  total: document.querySelector("#total"),
  count: document.querySelector("#count"),
  users: document.querySelector("#users"),
  dialog: document.querySelector("#expense-dialog"),
  form: document.querySelector("#expense-form"),
  formError: document.querySelector("#form-error")
}

let expenses = []

function fillCategories() {
  const options = Object.entries(categories).map(([value, title]) => `<option value="${value}">${title}</option>`).join("")
  document.querySelector("#filter-category").innerHTML = options
  document.querySelector("#category").innerHTML = options.replace('<option value="">Все категории</option>', "")
}

function filtersQuery() {
  const params = new URLSearchParams()
  const fields = { user_id: "filter-user", category: "filter-category", from: "filter-from", to: "filter-to" }
  Object.entries(fields).forEach(([key, id]) => {
    const value = document.querySelector(`#${id}`).value
    if (value) params.set(key, value)
  })
  return params.toString()
}

async function loadExpenses() {
  elements.status.textContent = "Загрузка..."
  try {
    const response = await fetch(`/api/expenses?${filtersQuery()}`)
    if (!response.ok) throw new Error("Не удалось загрузить расходы")
    expenses = await response.json()
    renderExpenses()
    elements.status.textContent = ""
  } catch (error) {
    elements.status.textContent = error.message
  }
}

function renderExpenses() {
  const money = new Intl.NumberFormat("ru-RU", { style: "currency", currency: "RUB" })
  elements.list.innerHTML = expenses.map(expense => `
    <tr>
      <td>${new Date(expense.spent_at).toLocaleDateString("ru-RU")}</td>
      <td><span class="user-pill">${expense.user_id}</span></td>
      <td>${categories[expense.category]}</td>
      <td>${escapeHTML(expense.comment)}</td>
      <td class="amount">${money.format(Number(expense.amount))}</td>
      <td class="actions"><button class="icon-button" data-edit="${expense.id}">Изменить</button><button class="icon-button danger" data-delete="${expense.id}">Удалить</button></td>
    </tr>`).join("")
  elements.empty.hidden = expenses.length > 0
  elements.total.textContent = money.format(expenses.reduce((sum, expense) => sum + Number(expense.amount), 0))
  elements.count.textContent = expenses.length
  elements.users.textContent = new Set(expenses.map(expense => expense.user_id)).size
}

function openForm(expense) {
  elements.form.reset()
  elements.formError.textContent = ""
  document.querySelector("#expense-id").value = expense?.id || ""
  document.querySelector("#form-title").textContent = expense ? "Изменить трату" : "Новая трата"
  document.querySelector("#user-id").value = expense?.user_id || ""
  document.querySelector("#user-id").disabled = Boolean(expense)
  document.querySelector("#amount").value = expense?.amount || ""
  document.querySelector("#category").value = expense?.category || "other"
  document.querySelector("#spent-at").value = expense ? expense.spent_at.slice(0, 10) : new Date().toISOString().slice(0, 10)
  document.querySelector("#comment").value = expense?.comment || ""
  elements.dialog.showModal()
}

async function saveExpense(event) {
  event.preventDefault()
  const id = document.querySelector("#expense-id").value
  const payload = {
    amount: document.querySelector("#amount").value,
    category: document.querySelector("#category").value,
    spent_at: document.querySelector("#spent-at").value,
    comment: document.querySelector("#comment").value.trim()
  }
  if (!id) payload.user_id = Number(document.querySelector("#user-id").value)
  const response = await fetch(id ? `/api/expenses/${id}` : "/api/expenses", {
    method: id ? "PUT" : "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(payload)
  })
  if (!response.ok) {
    const result = await response.json()
    elements.formError.textContent = result.error || "Не удалось сохранить трату"
    return
  }
  elements.dialog.close()
  await loadExpenses()
}

async function deleteExpense(id) {
  if (!confirm("Удалить эту трату?")) return
  const response = await fetch(`/api/expenses/${id}`, { method: "DELETE" })
  if (response.ok) await loadExpenses()
}

function escapeHTML(value) {
  const node = document.createElement("div")
  node.textContent = value
  return node.innerHTML
}

document.querySelector("#new-expense").addEventListener("click", () => openForm())
document.querySelector("#apply-filters").addEventListener("click", loadExpenses)
document.querySelector("#reset-filters").addEventListener("click", () => {
  document.querySelectorAll(".filters input, .filters select").forEach(input => input.value = "")
  loadExpenses()
})
document.querySelector("#close-dialog").addEventListener("click", () => elements.dialog.close())
document.querySelector("#cancel-dialog").addEventListener("click", () => elements.dialog.close())
elements.form.addEventListener("submit", saveExpense)
elements.list.addEventListener("click", event => {
  const editID = event.target.dataset.edit
  const deleteID = event.target.dataset.delete
  if (editID) openForm(expenses.find(expense => expense.id === Number(editID)))
  if (deleteID) deleteExpense(deleteID)
})

fillCategories()
loadExpenses()
