import test from "node:test";
import assert from "node:assert/strict";

import axios from "axios";
import MockAdapter from "axios-mock-adapter";

import * as telephone from "../../internal/generator/testdata/golden/telephone-js/index.js";
import * as dog from "../../internal/generator/testdata/golden/dog-parlor-js/index.js";
import * as booking from "../../internal/generator/testdata/golden/customer-booking-js/index.js";

function findOpBy(method, uri, sdkModule) {
    const routes = sdkModule.Endpoints();
    for (const [name, ep] of Object.entries(routes)) {
        if (ep.method === method && ep.uri === uri) return name;
    }
    return null;
}

function parseURL(config) {
    // axios-mock-adapter gives us full URL in config.url
    // Node's URL needs an absolute base; config.url should already be absolute
    return new URL(config.url);
}

test("telephone-js: GET /phones + auth header + query string", async () => {
    telephone.setBaseUrl("https://example.test");
    telephone.setTokenProvider(() => "TEST_TOKEN");

    const opName = findOpBy("GET", "/phones", telephone);
    assert.ok(opName, "Expected an operation for GET /phones");
    const fn = telephone[opName];
    assert.equal(typeof fn, "function", `Expected export function ${opName}`);

    const mock = new MockAdapter(axios);

    // Match any query ordering
    mock.onGet(/https:\/\/example\.test\/phones(\?.*)?$/).reply((config) => {
        assert.equal(config.method, "get");
        assert.equal(config.headers.Authorization, "Bearer TEST_TOKEN");

        const u = parseURL(config);
        assert.equal(u.pathname, "/phones");
        assert.equal(u.searchParams.get("country"), "US");
        assert.equal(u.searchParams.get("active"), "true");

        return [200, { items: [] }];
    });

    // stable signature: (path?, query?, body?, config?)
    const res = await fn(undefined, { country: "US", active: true });
    assert.equal(res.success, true, res.error || "expected success");
    assert.deepEqual(res.data, { items: [] });

    mock.restore();
});

test("dog-parlor-js: derived name works for GET /dogs/{id}/appointments", async () => {
    dog.setBaseUrl("https://example.test");
    dog.setTokenProvider(() => "");

    const opName = findOpBy("GET", "/dogs/{id}/appointments", dog);
    assert.ok(opName, "Expected an operation for GET /dogs/{id}/appointments");
    const fn = dog[opName];
    assert.equal(typeof fn, "function", `Expected export function ${opName}`);

    const mock = new MockAdapter(axios);

    mock.onGet("https://example.test/dogs/abc/appointments").reply((config) => {
        const u = parseURL(config);
        assert.equal(u.pathname, "/dogs/abc/appointments");
        return [200, { appointments: [{ date: "2026-02-05", service: "bath" }] }];
    });

    const res = await fn({ id: "abc" });
    assert.equal(res.success, true, res.error || "expected success");
    assert.equal(res.data.appointments.length, 1);

    mock.restore();
});

test("customer-booking-js: POST /customers/{customerId}/bookings sends body", async () => {
    booking.setBaseUrl("https://example.test");
    booking.setTokenProvider(() => "TOKEN");

    const opName = findOpBy("POST", "/customers/{customerId}/bookings", booking);
    assert.ok(opName, "Expected an operation for POST /customers/{customerId}/bookings");
    const fn = booking[opName];
    assert.equal(typeof fn, "function", `Expected export function ${opName}`);

    const mock = new MockAdapter(axios);

    mock.onPost("https://example.test/customers/c1/bookings").reply((config) => {
        assert.equal(config.headers.Authorization, "Bearer TOKEN");

        // axios usually sends JSON string; handle either
        const payload = typeof config.data === "string" ? JSON.parse(config.data) : config.data;
        assert.equal(payload.date, "2026-02-06");
        assert.equal(payload.notes, "trim nails");

        return [201, { id: "b1", date: payload.date, status: "created" }];
    });

    const res = await fn(
        { customerId: "c1" },
        undefined,
        { date: "2026-02-06", notes: "trim nails" }
    );

    assert.equal(res.success, true, res.error || "expected success");
    assert.equal(res.data.id, "b1");

    mock.restore();
});