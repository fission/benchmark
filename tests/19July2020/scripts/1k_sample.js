import http from "k6/http";
import { check } from "k6";

export let options = {
    stages: [
        { duration: "1m",  target: 100 }, // simulate ramp-up of traffic from 1 to 500 users over 1 minutes.
        { duration: "4m",  target: 1000 }, // ramp-up to 5000 users over 4 minutes (peak hour starts)
        { duration: "30m",  target: 1000 }, // Stay at 5K users for 30 min
        { duration: "1m",  target: 0 }, // stay at 500 users for short amount of time (peak hour)
    ],
    noConnectionReuse: true,
    noVUConnectionReuse: true,
    summaryTrendStats: [`avg`,`min`,`med`,`max`,`p(5)`,`p(10)`,`p(15)`,`p(20)`,`p(25)`,`p(30)`],
  };


export default function() {
    let params = { timeout: 30 }
    let res = http.get(`${__ENV.FN_ENDPOINT}`)
    check(res, {
        "status is 200": (r) => r.status === 200
    });
};
