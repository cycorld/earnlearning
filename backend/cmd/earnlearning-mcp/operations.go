package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
)

type operation struct {
	method, path                  string
	pathArgs, queryArgs, bodyArgs []string
	requiredBody                  []string
	mutation                      bool
}

func op(method, path string, paths, query, body []string) operation {
	return operation{method: method, path: path, pathArgs: paths, queryArgs: query, bodyArgs: body, mutation: method != http.MethodGet}
}

func required(s operation, fields ...string) operation {
	s.requiredBody = fields
	return s
}

var operations = map[string]operation{
	"health_get": op("GET", "/api/health", nil, nil, nil), "profile_get": op("GET", "/api/auth/me", nil, nil, nil),
	"companies_list": op("GET", "/api/companies", nil, nil, nil), "grants_list": op("GET", "/api/grants", nil, []string{"page", "limit", "status"}, nil),
	"posts_list":       op("GET", "/api/posts", nil, []string{"classroom_id", "page", "limit", "tag"}, nil),
	"admin_users_list": op("GET", "/api/admin/users", nil, nil, nil), "admin_pending_users_list": op("GET", "/api/admin/users/pending", nil, nil, nil),
	"admin_classroom_dashboard_get": op("GET", "/api/admin/classrooms/{classroom_id}/dashboard", []string{"classroom_id"}, nil, nil),
	"admin_milestones_list":         op("GET", "/api/admin/milestones", nil, nil, nil), "admin_tasks_list": op("GET", "/api/admin/tasks", nil, nil, nil),
	"admin_loans_list": op("GET", "/api/admin/loans", nil, []string{"status", "page", "limit"}, nil), "admin_disclosures_list": op("GET", "/api/admin/disclosures", nil, nil, nil),
	"classrooms_list": op("GET", "/api/classrooms", nil, nil, nil), "classroom_get": op("GET", "/api/classrooms/{id}", []string{"id"}, nil, nil),
	"wallet_get": op("GET", "/api/wallet", nil, nil, nil), "wallet_transactions_list": op("GET", "/api/wallet/transactions", nil, []string{"page", "limit", "tx_type", "start_date", "end_date"}, nil),
	"wallet_ranking_get": op("GET", "/api/wallet/ranking", nil, nil, nil), "company_get": op("GET", "/api/companies/{id}", []string{"id"}, nil, nil),
	"companies_mine_list": op("GET", "/api/companies/mine", nil, nil, nil), "company_services_list": op("GET", "/api/companies/{id}/services", []string{"id"}, nil, nil),
	"company_wallet_get": op("GET", "/api/companies/{id}/wallet", []string{"id"}, nil, nil), "company_transactions_list": op("GET", "/api/companies/{id}/transactions", []string{"id"}, []string{"page", "limit"}, nil),
	"grant_get": op("GET", "/api/grants/{id}", []string{"id"}, nil, nil), "post_get": op("GET", "/api/posts/{id}", []string{"id"}, nil, nil),
	"post_comments_list": op("GET", "/api/posts/{id}/comments", []string{"id"}, nil, nil), "milestones_mine_get": op("GET", "/api/milestones/mine", nil, nil, nil),
	"milestone_files_list": op("GET", "/api/milestones/files", nil, []string{"type"}, nil), "notifications_list": op("GET", "/api/notifications", nil, []string{"page", "limit", "is_read", "type"}, nil),
	"investment_rounds_list": op("GET", "/api/investment/rounds", nil, []string{"company_id", "status", "page", "limit"}, nil), "investment_portfolio_get": op("GET", "/api/investment/portfolio", nil, nil, nil),

	"admin_user_approve": op("PUT", "/api/admin/users/{id}/approve", []string{"id"}, nil, nil), "admin_user_reject": op("PUT", "/api/admin/users/{id}/reject", []string{"id"}, nil, nil),
	"admin_milestone_approve": op("POST", "/api/admin/milestones/{id}/approve", []string{"id"}, nil, []string{"admin_note"}), "admin_milestone_reject": op("POST", "/api/admin/milestones/{id}/reject", []string{"id"}, nil, []string{"admin_note"}),
	"admin_grant_application_approve": op("POST", "/api/admin/grants/{grant_id}/approve/{application_id}", []string{"grant_id", "application_id"}, nil, nil),
	"admin_grant_application_revoke":  op("POST", "/api/admin/grants/{grant_id}/revoke/{application_id}", []string{"grant_id", "application_id"}, nil, nil),
	"admin_disclosure_approve":        required(op("POST", "/api/admin/disclosures/{id}/approve", []string{"id"}, nil, []string{"reward", "admin_note"}), "reward"), "admin_disclosure_reject": op("POST", "/api/admin/disclosures/{id}/reject", []string{"id"}, nil, []string{"admin_note"}),
	"admin_loan_approve": required(op("PUT", "/api/admin/loans/{id}/approve", []string{"id"}, nil, []string{"interest_rate"}), "interest_rate"), "admin_loan_reject": op("PUT", "/api/admin/loans/{id}/reject", []string{"id"}, nil, nil),
	"company_service_validate":       op("POST", "/api/companies/{company_id}/services/{service_id}/validate", []string{"company_id", "service_id"}, nil, nil),
	"company_service_rybbit_connect": op("POST", "/api/companies/{company_id}/services/{service_id}/rybbit/connect", []string{"company_id", "service_id"}, nil, nil),

	"post_create":             required(op("POST", "/api/channels/{channel_id}/posts", []string{"channel_id"}, nil, []string{"content", "post_type", "media", "tags"}), "content"),
	"post_update":             required(op("PUT", "/api/posts/{id}", []string{"id"}, nil, []string{"content", "tags", "channel_id"}), "content"),
	"comment_create":          required(op("POST", "/api/posts/{id}/comments", []string{"id"}, nil, []string{"content", "media"}), "content"),
	"assignment_submit":       required(op("POST", "/api/assignments/{id}/submit", []string{"id"}, nil, []string{"content", "files"}), "content"),
	"company_create":          required(op("POST", "/api/companies", nil, nil, []string{"name", "description", "logo_url", "initial_capital"}), "name", "initial_capital"),
	"company_update":          op("PUT", "/api/companies/{id}", []string{"id"}, nil, []string{"name", "description", "logo_url", "service_url"}),
	"company_service_create":  required(op("POST", "/api/companies/{id}/services", []string{"id"}, nil, []string{"name", "url"}), "name", "url"),
	"company_service_update":  required(op("PUT", "/api/companies/{company_id}/services/{service_id}", []string{"company_id", "service_id"}, nil, []string{"name", "url"}), "name", "url"),
	"grant_apply":             required(op("POST", "/api/grants/{id}/apply", []string{"id"}, nil, []string{"proposal"}), "proposal"),
	"investment_round_create": required(op("POST", "/api/investment/rounds", nil, nil, []string{"company_id", "target_amount", "offered_percent"}), "company_id", "target_amount", "offered_percent"),
	"investment_create":       required(op("POST", "/api/investment/rounds/{id}/invest", []string{"id"}, nil, []string{"shares"}), "shares"),
	"exchange_order_create":   required(op("POST", "/api/exchange/orders", nil, nil, []string{"company_id", "order_type", "shares", "price"}), "company_id", "order_type", "shares", "price"),
	"wallet_transfer":         required(op("POST", "/api/wallet/transfer", nil, nil, []string{"target_user_id", "target_type", "amount", "description"}), "target_user_id", "target_type", "amount"),
	"company_transfer":        required(op("POST", "/api/companies/{id}/transfer", []string{"id"}, nil, []string{"target_id", "target_type", "amount", "description"}), "target_id", "target_type", "amount"),
	"loan_apply":              required(op("POST", "/api/loans", nil, nil, []string{"amount", "purpose"}), "amount", "purpose"), "loan_repay": required(op("POST", "/api/loans/{id}/repay", []string{"id"}, nil, []string{"amount"}), "amount"),
}

func operationTools() []tool {
	names := make([]string, 0, len(operations))
	for name := range operations {
		names = append(names, name)
	}
	sort.Strings(names)
	out := make([]tool, 0, len(names))
	for _, name := range names {
		s := operations[name]
		props := map[string]any{}
		required := []string{}
		for _, key := range s.pathArgs {
			props[key] = map[string]any{"type": "integer", "minimum": 1}
			required = append(required, key)
		}
		for _, key := range s.queryArgs {
			props[key] = fieldSchema(key)
		}
		for _, key := range s.bodyArgs {
			props[key] = fieldSchema(key)
		}
		required = append(required, s.requiredBody...)
		if s.mutation {
			props["confirm"] = map[string]any{"type": "string", "const": name}
			required = append(required, "confirm")
		}
		schema := map[string]any{"type": "object", "properties": props, "additionalProperties": false}
		if len(required) > 0 {
			schema["required"] = required
		}
		out = append(out, tool{name, "기존 EarnLearning API " + s.method + " " + s.path + " 호출", schema})
	}
	return out
}

func fieldSchema(key string) map[string]any {
	switch key {
	case "target_type":
		return map[string]any{"type": "string", "enum": []string{"user", "company"}}
	case "order_type":
		return map[string]any{"type": "string", "enum": []string{"buy", "sell"}}
	case "is_read":
		return map[string]any{"type": "boolean"}
	case "offered_percent", "interest_rate":
		return map[string]any{"type": "number", "minimum": 0}
	case "media", "files":
		return map[string]any{"type": "string", "maxLength": 10000}
	}
	if key == "page" || key == "limit" || key == "amount" || key == "price" || key == "shares" || key == "reward" || key == "initial_capital" || key == "target_amount" || strings.HasSuffix(key, "_id") {
		return map[string]any{"type": "integer", "minimum": 1}
	}
	return map[string]any{"type": "string", "maxLength": 10000}
}

func (c *apiClient) callOperation(name string, args map[string]any) (json.RawMessage, error) {
	s, ok := operations[name]
	if !ok {
		return nil, fmt.Errorf("unknown tool %q", name)
	}
	if args == nil {
		args = map[string]any{}
	}
	allowed := append(append(append([]string{}, s.pathArgs...), s.queryArgs...), s.bodyArgs...)
	if s.mutation {
		allowed = append(allowed, "confirm")
	}
	if err := validateArgs(args, allowed...); err != nil {
		return nil, err
	}
	if s.mutation && args["confirm"] != name {
		return nil, fmt.Errorf("confirm must exactly equal %q", name)
	}
	for _, key := range s.requiredBody {
		if _, ok := args[key]; !ok {
			return nil, fmt.Errorf("%s is required", key)
		}
	}
	path := s.path
	for _, key := range s.pathArgs {
		n, err := intArg(args, key, true, 1, 1_000_000_000, 0)
		if err != nil {
			return nil, err
		}
		path = strings.ReplaceAll(path, "{"+key+"}", strconv.Itoa(n))
	}
	q := url.Values{}
	for _, key := range s.queryArgs {
		v, exists := args[key]
		if !exists {
			continue
		}
		if key == "page" || key == "limit" || strings.HasSuffix(key, "_id") {
			max := 1_000_000_000
			if key == "limit" {
				max = maxResults
			}
			n, err := intArg(args, key, true, 1, max, 0)
			if err != nil {
				return nil, err
			}
			q.Set(key, strconv.Itoa(n))
		} else if flag, ok := v.(bool); ok {
			q.Set(key, strconv.FormatBool(flag))
		} else {
			text, ok := v.(string)
			if !ok || len(text) > 100 {
				return nil, fmt.Errorf("%s must be a string of at most 100 characters", key)
			}
			q.Set(key, text)
		}
	}
	var body io.Reader
	if len(s.bodyArgs) > 0 {
		values := map[string]any{}
		for _, key := range s.bodyArgs {
			if v, exists := args[key]; exists {
				values[key] = v
			}
		}
		encoded, err := json.Marshal(values)
		if err != nil {
			return nil, errors.New("request body could not be encoded")
		}
		body = bytes.NewReader(encoded)
	}
	u := *c.base
	u.Path += path
	u.RawQuery = q.Encode()
	req, err := http.NewRequestWithContext(context.Background(), s.method, u.String(), body)
	if err != nil {
		return nil, errors.New("EarnLearning API request could not be created")
	}
	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Accept", "application/json")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := c.client.Do(req)
	if err != nil {
		return nil, errors.New("EarnLearning API request failed")
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseBytes+1))
	if err != nil {
		return nil, errors.New("EarnLearning API response could not be read")
	}
	if len(data) > maxResponseBytes {
		return nil, errors.New("EarnLearning API response is too large")
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("EarnLearning API returned HTTP %d", resp.StatusCode)
	}
	if len(bytes.TrimSpace(data)) == 0 {
		return json.RawMessage("null"), nil
	}
	if !json.Valid(data) {
		return nil, errors.New("EarnLearning API returned invalid JSON")
	}
	return data, nil
}
