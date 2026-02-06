import test from "node:test";
import assert from "node:assert/strict";
import MockAdapter from "axios-mock-adapter";

import * as telephone from "../../internal/generator/testdata/golden/telephone-js/index.js";
import { axios as telephoneAxios } from "../../internal/generator/testdata/golden/telephone-js/requests.js";

import * as dog from "../../internal/generator/testdata/golden/dog-parlor-js/index.js";
import { axios as dogAxios } from "../../internal/generator/testdata/golden/dog-parlor-js/requests.js";

import * as booking from "../../internal/generator/testdata/golden/customer-booking-js/index.js";
import { axios as bookingAxios } from "../../internal/generator/testdata/golden/customer-booking-js/requests.js";

function findEndpointBy(method, uri, sdkModule) {
    const routes = sdkModule.Endpoints();
    for (const [name, ep] of Object.entries(routes)) {
        if (ep.method === method && ep.uri === uri) return { name, ep };
    }
    return null;
}

function addUnmatchedFallback(mock, label) {
    mock.onAny().reply((config) => {
        throw new Error(`[${label}] Unmatched request: ${config.method?.toUpperCase()} ${config.url}`);
    });
}

test("telephone-js: GET /phones mocked", async () => {
    telephone.setBaseUrl("https://example.test");
    telephone.setTokenProvider(() => "TEST_TOKEN");

    const found = findEndpointBy("GET", "/phones", telephone);
    assert.ok(found, "Expected GET /phones");
    const { name: opName } = found;

    const fn = telephone[opName];
    assert.equal(typeof fn, "function");

    const mock = new MockAdapter(telephoneAxios);

    mock.onGet(/https:\/\/example\.test\/phones(\?.*)?$/).reply((config) => {
        assert.equal(config.headers.Authorization, "Bearer TEST_TOKEN");

        const u = new URL(config.url);
        assert.equal(u.pathname, "/phones");
        assert.equal(u.searchParams.get("country"), "US");
        assert.equal(u.searchParams.get("active"), "true");
        return [200, { items: [] }];
    });

    addUnmatchedFallback(mock, "telephone");

    const res = await fn(undefined, { country: "US", active: true }, undefined, { __debug: true });
    assert.equal(res.success, true);
    assert.deepEqual(res.data, { items: [] });

    mock.restore();
});

test("dog-parlor-js: GET /dogs/{id}/appointments mocked", async () => {
    dog.setBaseUrl("https://example.test");
    dog.setTokenProvider(() => "");

    const found = findEndpointBy("GET", "/dogs/{id}/appointments", dog);
    assert.ok(found, "Expected GET /dogs/{id}/appointments");
    const { name: opName } = found;

    const fn = dog[opName];
    assert.equal(typeof fn, "function");

    const mock = new MockAdapter(dogAxios);

    mock.onGet("https://example.test/dogs/abc/appointments").reply(200, {
        appointments: [{ date: "2026-02-05", service: "bath" }],
    });

    addUnmatchedFallback(mock, "dog-parlor");

    const res = await fn({ id: "abc" }, undefined, undefined, { __debug: true });
    assert.equal(res.success, true);

    mock.restore();
});

test("customer-booking-js: POST /customers/{customerId}/bookings mocked", async () => {
    booking.setBaseUrl("https://example.test");
    booking.setTokenProvider(() => "TOKEN");

    const found = findEndpointBy("POST", "/customers/{customerId}/bookings", booking);
    assert.ok(found, "Expected POST /customers/{customerId}/bookings");
    const { name: opName } = found;

    const fn = booking[opName];
    assert.equal(typeof fn, "function");

    const mock = new MockAdapter(bookingAxios);

    mock.onPost("https://example.test/customers/c1/bookings").reply((config) => {
        assert.equal(config.headers.Authorization, "Bearer TOKEN");
        const payload = typeof config.data === "string" ? JSON.parse(config.data) : config.data;
        assert.equal(payload.date, "2026-02-06");
        assert.equal(payload.notes, "trim nails");
        return [201, { id: "b1" }];
    });


    addUnmatchedFallback(mock, "customer-booking");

    const res = await fn(
        { customerId: "c1" },
        undefined,
        { date: "2026-02-06", notes: "trim nails" },
        { __debug: true }
    );

    console.log(res)

    assert.equal(res.success, true);
    assert.equal(res.data.id, "b1");

    mock.restore();
});