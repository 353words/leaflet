SELECT
    strftime('%H:%M', time),
    AVG(lat),
    AVG(lng)
FROM points
    GROUP BY strftime('%H:%M', time)
;
