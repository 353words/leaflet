// This script relies on HTML having a "points" and "center" variables.
function on_loaded() {
    // Create map & tiles.
    var map = L.map('map').setView(center, 15);
    L.tileLayer(
        'https://tile.openstreetmap.org/{z}/{x}/{y}.png',
        {
            maxZoom: 19,
            attribution: '&copy; <a href="http://www.openstreetmap.org/copyright">OpenStreetMap</a>'
        }
    ).addTo(map);

    // Add points with tooltip to map.
    points.forEach((pt) => {
        let circle = L.circle(
            [pt.lat, pt.lng],
            {
                color: 'red',
                radius: 20
            }).addTo(map);
        circle.bindPopup(pt.time);
    });
}

document.addEventListener('DOMContentLoaded', on_loaded);
