migrate((app) => {
    const users = new Collection({
        type: "auth",
        name: "members",
        listRule: "id = @request.auth.id",
        viewRule: "id = @request.auth.id",
        createRule: "",
        updateRule: "id = @request.auth.id",
        deleteRule: "id = @request.auth.id",
        fields: [
            { type: "text", name: "name", required: true, min: 2, max: 80 },
            { type: "file", name: "avatar", maxSelect: 1, maxSize: 5242880, mimeTypes: ["image/jpeg", "image/png", "image/webp"] }
        ],
        passwordAuth: { enabled: true, identityFields: ["email"] }
    })
    app.save(users)

    const groups = new Collection({
        type: "base",
        name: "groups",
        listRule: "owner = @request.auth.id || members.id ?= @request.auth.id",
        viewRule: "owner = @request.auth.id || members.id ?= @request.auth.id",
        createRule: "@request.body.owner = @request.auth.id && @request.body.members ?= @request.auth.id",
        updateRule: "owner = @request.auth.id && @request.body.owner:changed = false",
        deleteRule: "owner = @request.auth.id",
        fields: [
            { type: "text", name: "name", required: true, min: 2, max: 80 },
            { type: "text", name: "description", max: 300 },
            { type: "text", name: "color", required: true, max: 20 },
            { type: "select", name: "currency", required: true, maxSelect: 1, values: ["TWD", "USD", "JPY", "EUR"] },
            { type: "relation", name: "owner", required: true, maxSelect: 1, collectionId: users.id, cascadeDelete: true },
            { type: "relation", name: "members", required: true, maxSelect: 50, collectionId: users.id, cascadeDelete: false }
        ],
        indexes: ["CREATE INDEX idx_groups_owner ON groups (owner)"]
    })
    app.save(groups)

    const subscriptions = new Collection({
        type: "base",
        name: "subscriptions",
        listRule: "group.owner = @request.auth.id || group.members.id ?= @request.auth.id",
        viewRule: "group.owner = @request.auth.id || group.members.id ?= @request.auth.id",
        createRule: "@request.auth.id != '' && @collection.groups.id ?= @request.body.group && (@collection.groups.owner = @request.auth.id || @collection.groups.members.id ?= @request.auth.id)",
        updateRule: "(group.owner = @request.auth.id || group.members.id ?= @request.auth.id) && @request.body.group:changed = false",
        deleteRule: "group.owner = @request.auth.id || group.members.id ?= @request.auth.id",
        fields: [
            { type: "relation", name: "group", required: true, maxSelect: 1, collectionId: groups.id, cascadeDelete: true },
            { type: "text", name: "name", required: true, min: 2, max: 100 },
            { type: "text", name: "category", required: true, max: 40 },
            { type: "number", name: "amount", required: true, min: 0 },
            { type: "select", name: "currency", required: true, maxSelect: 1, values: ["TWD", "USD", "JPY", "EUR"] },
            { type: "select", name: "billing_cycle", required: true, maxSelect: 1, values: ["monthly", "quarterly", "yearly"] },
            { type: "date", name: "next_billing", required: true },
            { type: "select", name: "status", required: true, maxSelect: 1, values: ["active", "paused", "cancelled"] },
            { type: "text", name: "notes", max: 500 }
        ],
        indexes: ["CREATE INDEX idx_subscriptions_group ON subscriptions (`group`)", "CREATE INDEX idx_subscriptions_next_billing ON subscriptions (next_billing)"]
    })
    app.save(subscriptions)

    const expenses = new Collection({
        type: "base",
        name: "expenses",
        listRule: "group.owner = @request.auth.id || group.members.id ?= @request.auth.id",
        viewRule: "group.owner = @request.auth.id || group.members.id ?= @request.auth.id",
        createRule: "@request.auth.id != '' && @collection.groups.id ?= @request.body.group && (@collection.groups.owner = @request.auth.id || @collection.groups.members.id ?= @request.auth.id)",
        updateRule: "(group.owner = @request.auth.id || group.members.id ?= @request.auth.id) && @request.body.group:changed = false",
        deleteRule: "group.owner = @request.auth.id || group.members.id ?= @request.auth.id",
        fields: [
            { type: "relation", name: "group", required: true, maxSelect: 1, collectionId: groups.id, cascadeDelete: true },
            { type: "text", name: "title", required: true, min: 2, max: 100 },
            { type: "number", name: "amount", required: true, min: 0 },
            { type: "relation", name: "paid_by", required: true, maxSelect: 1, collectionId: users.id, cascadeDelete: false },
            { type: "date", name: "expense_date", required: true },
            { type: "text", name: "category", required: true, max: 40 },
            { type: "text", name: "notes", max: 500 }
        ],
        indexes: ["CREATE INDEX idx_expenses_group ON expenses (`group`)", "CREATE INDEX idx_expenses_date ON expenses (expense_date)"]
    })
    app.save(expenses)
}, (app) => {
    for (const name of ["expenses", "subscriptions", "groups", "members"]) {
        try {
            app.delete(app.findCollectionByNameOrId(name))
        } catch (_) {
            // Collection may already be absent during a partial rollback.
        }
    }
})
