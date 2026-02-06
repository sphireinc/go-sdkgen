import test from "node:test";
import assert from "node:assert/strict";

import axios from "axios";
import MockAdapter from "axios-mock-adapter";

import * as telephone from "TelephoneSDK-js";
import * as dog from "DogParlorSDK-js";
import * as booking from "CustomerBookingSDK-js";

function findOpBy(method, uri, sdkModule) {
    const routes = sdkModule.Endpoints();
    for (const [name, ep] of Object.entries(routes)) {
        if (ep.method === method && ep.uri === uri) return name;
    }
    return null;
}

test("telephone-js: GET /phones + auth header + query string", async () => {
    telephone.setBaseUrl("https://example.test");
    telephone.setTokenProvider(() => "TEST_TOKEN");

    const opName = findOpBy("GET", "/phones", telephone);
    assert.ok(opName, "Expected an operation for GET /phones");

    const fn = telephone[opName];
    assert.equal(typeof fn, "function", `Expected export function ${opName}`);

    const mock = new MockAdapter(axios);

    mock.onGet("https://example.test/phones?country=US&active=true").reply((config) => {
        assert.equal(config.method, "get");
        assert.equal(config.headers.Authorization, "Bearer TEST_TOKEN");
        return [200, { items: [] }];
    });

    // signature: (path?, query?, body?, config?)
    const res = await fn(undefined, { country: "US", active: true });
    assert.equal(res.success, true);
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

    mock.onGet("https://example.test/dogs/abc/appointments").reply(200, {
        appointments: [{ date: "2026-02-05", service: "bath" }],
    });

    const res = await fn({ id: "abc" });
    assert.equal(res.success, true);
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
        assert.equal(config.method, "post");
        assert.equal(config.headers.Authorization, "Bearer TOKEN");

        const parsed = JSON.parse(config.data);
        assert.equal(parsed.date, "2026-02-06");
        assert.equal(parsed.notes, "trim nails");
        return [201, { id: "b1", date: parsed.date, status: "created" }];
    });

    const res = await fn(
        { customerId: "c1" },
        undefined,
        { date: "2026-02-06", notes: "trim nails" }
    );

    assert.equal(res.success, true);
    assert.equal(res.data.id, "b1");

    mock.restore();
});