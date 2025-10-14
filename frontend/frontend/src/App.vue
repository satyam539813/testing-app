<template>
  <div class="app">
    <h1>🧭 Wanderwise Planner</h1>

    <form @submit.prevent="getRoute">
      <input v-model="source" type="text" placeholder="Enter source..." required />
      <input v-model="destination" type="text" placeholder="Enter destination..." required />
      <input v-model="budget" type="number" placeholder="Enter budget (₹)" required />
      <button type="submit" :disabled="loading">Generate Plan</button>
    </form>

    <div id="map"></div>

    <div v-if="loading" class="loading">Generating plan...</div>

    <div v-if="reply" class="response">
      <h3>Travel Plan from {{ reply.source }} to {{ reply.destination }}</h3>
      <p><strong>Budget:</strong> ₹{{ reply.budget }}</p>
      <h4>Day-wise Itinerary:</h4>
      <ul>
        <li v-for="day in reply.days" :key="day.day">
          <strong>Day {{ day.day }}:</strong> {{ day.activities }} - <strong>Expenses:</strong>
          <ul>
            <li v-for="(expense, category) in day.expenses" :key="category">
              {{ category }}: ₹{{ expense }}
            </li>
          </ul>
        </li>
      </ul>
    </div>
  </div>
</template>

<script setup>
import { ref, onMounted } from "vue";
import L from "leaflet";

const source = ref("");
const destination = ref("");
const budget = ref("");
const reply = ref(null);
const loading = ref(false);
let map, routeLayer;

onMounted(() => {
  map = L.map("map").setView([28.6139, 77.2090], 6); // Default: India
  L.tileLayer("https://{s}.tile.openstreetmap.org/{z}/{x}/{y}.png", {
    attribution: "&copy; OpenStreetMap contributors",
  }).addTo(map);
});

const getCoords = async (place) => {
  const res = await fetch(`https://nominatim.openstreetmap.org/search?format=json&q=${place}`);
  const data = await res.json();
  if (data.length === 0) throw new Error("Location not found: " + place);
  return [parseFloat(data[0].lat), parseFloat(data[0].lon)];
};

const getRoute = async () => {
  loading.value = true;
  try {
    const [srcCoords, destCoords] = await Promise.all([
      getCoords(source.value),
      getCoords(destination.value),
    ]);

    if (routeLayer) map.removeLayer(routeLayer);
    routeLayer = L.polyline([srcCoords, destCoords], { color: "blue" }).addTo(map);
    map.fitBounds(routeLayer.getBounds());

    const res = await fetch("http://localhost:8080/api/route", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({
        source: source.value,
        destination: destination.value,
        budget: parseFloat(budget.value),
      }),
    });

    if (!res.ok) throw new Error("Failed to fetch backend response");

    const data = await res.json();
    reply.value = data;
  } catch (err) {
    alert("❌ " + err.message);
  } finally {
    loading.value = false;
  }
};
</script>

<style>
.app {
  max-width: 600px;
  margin: 50px auto;
  text-align: center;
  font-family: "Gilroy", sans-serif;
}
input {
  width: 40%;
  padding: 10px;
  margin: 5px;
  border-radius: 6px;
  border: 1px solid #ccc;
}
button {
  padding: 10px 15px;
  background: #0a84ff;
  border: none;
  color: white;
  cursor: pointer;
  border-radius: 6px;
}
button:disabled {
  background: #ccc;
  cursor: not-allowed;
}
.loading {
  margin-top: 20px;
  font-size: 18px;
  color: #0a84ff;
}
#map {
  height: 400px;
  margin-top: 20px;
  border-radius: 10px;
  overflow: hidden;
}
.response {
  margin-top: 20px;
  background: #f1f1f1;
  padding: 15px;
  border-radius: 8px;
  text-align: left;
  white-space: pre-wrap;} 
  </style>