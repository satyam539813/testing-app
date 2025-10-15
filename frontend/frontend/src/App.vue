 <template>
  <div class="app">
    <!-- Header Section -->
    <header class="header">
      <div class="header-content">
        <h1 class="title">
          <span class="icon">✈️</span>
          Wanderwise Planner
        </h1>
        <p class="subtitle">Plan your perfect journey with AI-powered itineraries</p>
      </div>
    </header>

    <!-- Search Form -->
    <div class="search-container">
      <form @submit.prevent="getRoute" class="search-form">
        <div class="input-group">
          <div class="input-wrapper">
            <span class="input-icon">📍</span>
            <input 
              v-model="source" 
              type="text" 
              placeholder="From (e.g., Delhi)" 
              required 
              class="input-field"
            />
          </div>
          
          <div class="input-wrapper">
            <span class="input-icon">🎯</span>
            <input 
              v-model="destination" 
              type="text" 
              placeholder="To (e.g., Goa)" 
              required 
              class="input-field"
            />
          </div>
          
          <div class="input-wrapper">
            <span class="input-icon">💰</span>
            <input 
              v-model="budget" 
              type="number" 
              placeholder="Budget (₹)" 
              required 
              min="1" 
              class="input-field"
            />
          </div>
        </div>
        
        <button type="submit" :disabled="loading" class="submit-btn">
          <span v-if="!loading">Generate Itinerary</span>
          <span v-else class="loading-text">
            <span class="spinner-small"></span>
            Generating...
          </span>
        </button>
      </form>
    </div>

    <!-- Map Section -->
    <div class="map-container">
      <div id="map" class="map"></div>
    </div>

    <!-- Loading State -->
    <transition name="fade">
      <div v-if="loading" class="loading-overlay">
        <div class="loading-card">
          <div class="loading-spinner"></div>
          <h3>Crafting Your Perfect Journey</h3>
          <p>Our AI is analyzing the best routes and experiences...</p>
        </div>
      </div>
    </transition>

    <!-- Error State -->
    <transition name="slide-down">
      <div v-if="error" class="error-banner">
        <span class="error-icon">⚠️</span>
        <span>{{ error }}</span>
        <button @click="error = null" class="close-btn">✕</button>
      </div>
    </transition>

    <!-- Travel Plan Results -->
    <transition name="fade">
      <div v-if="reply" class="results-container">
        <div class="results-header">
          <h2 class="results-title">
            <span class="route-badge">{{ reply.source }}</span>
            <span class="arrow">→</span>
            <span class="route-badge">{{ reply.destination }}</span>
          </h2>
          <div class="budget-display">
            <span class="budget-label">Total Budget</span>
            <span class="budget-amount">₹{{ reply.budget.toLocaleString('en-IN') }}</span>
          </div>
        </div>

        <!-- Day Cards -->
        <div class="days-grid">
          <div 
            v-for="(day, index) in reply.days" 
            :key="day.day"
            class="day-card"
            :style="{ animationDelay: `${index * 0.1}s` }"
          >
            <div class="day-header">
              <div class="day-number">
                <span class="day-label">Day</span>
                <span class="day-value">{{ day.day }}</span>
              </div>
              <div class="day-total">
                <span class="total-label">Daily Cost</span>
                <span class="total-amount">₹{{ calculateDayTotal(day.expenses) }}</span>
              </div>
            </div>

            <div class="day-content">
              <div class="activities-section">
                <h4 class="section-title">
                  <span class="section-icon">🗓️</span>
                  Activities
                </h4>
                <div class="activities-text">
                  <p v-if="typeof day.activities === 'string'">{{ day.activities }}</p>
                  <ul v-else-if="Array.isArray(day.activities)">
                    <li v-for="activity in day.activities" :key="activity">{{ activity }}</li>
                  </ul>
                  <p v-else>{{ day.activities }}</p>
                </div>
              </div>

              <div class="expenses-section">
                <h4 class="section-title">
                  <span class="section-icon">💳</span>
                  Expenses Breakdown
                </h4>
                <div class="expenses-list">
                  <div 
                    v-for="(expense, category) in day.expenses" 
                    :key="category"
                    class="expense-item"
                  >
                    <span class="expense-category">{{ category }}</span>
                    <span class="expense-dots"></span>
                    <span class="expense-amount">₹{{ formatExpense(expense) }}</span>
                  </div>
                </div>
              </div>
            </div>
          </div>
        </div>

        <!-- Summary Section -->
        <div class="summary-card">
          <h3 class="summary-title">Trip Summary</h3>
          <div class="summary-stats">
            <div class="stat-item">
              <span class="stat-icon">📅</span>
              <span class="stat-label">Duration</span>
              <span class="stat-value">{{ reply.days.length }} Days</span>
            </div>
            <div class="stat-item">
              <span class="stat-icon">💰</span>
              <span class="stat-label">Total Budget</span>
              <span class="stat-value">₹{{ reply.budget.toLocaleString('en-IN') }}</span>
            </div>
            <div class="stat-item">
              <span class="stat-icon">🎯</span>
              <span class="stat-label">Route</span>
              <span class="stat-value">{{ reply.source }} → {{ reply.destination }}</span>
            </div>
          </div>
        </div>
      </div>
    </transition>
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
const error = ref(null);
let map, routeLayer, markers = [];

onMounted(() => {
  try {
    map = L.map("map").setView([20.5937, 78.9629], 5);
    L.tileLayer("https://{s}.tile.openstreetmap.org/{z}/{x}/{y}.png", {
      attribution: "&copy; OpenStreetMap contributors",
      maxZoom: 18,
    }).addTo(map);
  } catch (err) {
    console.error("Failed to initialize map:", err);
    error.value = "Failed to initialize map. Please refresh the page.";
  }
});

const formatExpense = (expense) => {
  if (typeof expense === 'number') {
    return expense.toLocaleString('en-IN');
  }
  return expense;
};

const calculateDayTotal = (expenses) => {
  if (!expenses) return '0';
  const total = Object.values(expenses).reduce((sum, expense) => {
    const amount = typeof expense === 'number' ? expense : parseFloat(expense) || 0;
    return sum + amount;
  }, 0);
  return total.toLocaleString('en-IN');
};

const getCoords = async (place) => {
  const res = await fetch(
    `https://nominatim.openstreetmap.org/search?format=json&q=${encodeURIComponent(place)}`
  );
  const data = await res.json();
  if (data.length === 0) throw new Error(`Location not found: ${place}`);
  return [parseFloat(data[0].lat), parseFloat(data[0].lon)];
};

const clearMap = () => {
  if (routeLayer && map) {
    map.removeLayer(routeLayer);
  }
  markers.forEach(marker => map.removeLayer(marker));
  markers = [];
};

const getRoute = async () => {
  loading.value = true;
  error.value = null;
  reply.value = null;

  try {
    const [srcCoords, destCoords] = await Promise.all([
      getCoords(source.value),
      getCoords(destination.value),
    ]);

    clearMap();
    
    if (map) {
      routeLayer = L.polyline([srcCoords, destCoords], { 
        color: "#6366f1",
        weight: 4,
        opacity: 0.8,
        smoothFactor: 1
      }).addTo(map);
      
      const srcMarker = L.marker(srcCoords, {
        icon: L.divIcon({
          className: 'custom-marker',
          html: '<div style="background: #10b981; width: 30px; height: 30px; border-radius: 50%; border: 3px solid white; box-shadow: 0 2px 8px rgba(0,0,0,0.3);"></div>',
          iconSize: [30, 30]
        })
      }).addTo(map).bindPopup(`<b>Start:</b> ${source.value}`);
      
      const destMarker = L.marker(destCoords, {
        icon: L.divIcon({
          className: 'custom-marker',
          html: '<div style="background: #ef4444; width: 30px; height: 30px; border-radius: 50%; border: 3px solid white; box-shadow: 0 2px 8px rgba(0,0,0,0.3);"></div>',
          iconSize: [30, 30]
        })
      }).addTo(map).bindPopup(`<b>Destination:</b> ${destination.value}`);
      
      markers.push(srcMarker, destMarker);
      map.fitBounds(routeLayer.getBounds(), { padding: [50, 50] });
    }

    const res = await fetch("http://localhost:8080/api/route", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({
        source: source.value,
        destination: destination.value,
        budget: parseFloat(budget.value),
      }),
    });

    if (!res.ok) {
      const errorData = await res.json().catch(() => ({ error: "Server error" }));
      throw new Error(errorData.error || `Server responded with status ${res.status}`);
    }

    const data = await res.json();
    
    if (!data.days || data.days.length === 0) {
      throw new Error("No travel plan was generated");
    }
    
    reply.value = data;
  } catch (err) {
    console.error("Error:", err);
    error.value = err.message;
  } finally {
    loading.value = false;
  }
};
</script>

<style scoped>
* {
  margin: 0;
  padding: 0;
  box-sizing: border-box;
}

.app {
  min-height: 100vh;
  background: black;
  padding-bottom: 60px;
}

/* Header Section */
.header {
  background: black;
  backdrop-filter: blur(10px);
  padding: 40px 20px;
  text-align: center;
  color: white;
  box-shadow: 0 4px 20px rgba(0, 0, 0, 0.1);
}

.header-content {
  max-width: 800px;
  margin: 0 auto;
}

.title {
  font-size: 3rem;
  font-weight: 800;
  margin-bottom: 10px;
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 15px;
}

.icon {
  font-size: 3.5rem;
  animation: float 3s ease-in-out infinite;
}

@keyframes float {
  0%, 100% { transform: translateY(0px); }
  50% { transform: translateY(-10px); }
}

.subtitle {
  font-size: 1.2rem;
  opacity: 0.9;
  font-weight: 300;
}

/* Search Container */
.search-container {
  max-width: 900px;
  margin: -30px auto 40px;
  padding: 0 20px;
  position: relative;
  z-index: 10;
}

.search-form {
  background: white;
  padding: 30px;
  border-radius: 20px;
  box-shadow: 0 20px 60px rgba(0, 0, 0, 0.3);
}

.input-group {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(250px, 1fr));
  gap: 15px;
  margin-bottom: 20px;
}

.input-wrapper {
  position: relative;
}

.input-icon {
  position: absolute;
  left: 15px;
  top: 50%;
  transform: translateY(-50%);
  font-size: 1.3rem;
  pointer-events: none;
}

.input-field {
  width: 100%;
  padding: 15px 15px 15px 50px;
  border: 2px solid #e5e7eb;
  border-radius: 12px;
  font-size: 16px;
  transition: all 0.3s;
  background: #f9fafb;
}

.input-field:focus {
  outline: none;
  border-color: #667eea;
  background: white;
  box-shadow: 0 0 0 4px rgba(102, 126, 234, 0.1);
}

.submit-btn {
  width: 100%;
  padding: 16px;
  background: black;
  border-radius: 90px;
  color: white;
  border: none;
  border-radius: 12px;
  font-size: 18px;
  font-weight: 600;
  cursor: pointer;
  transition: all 0.3s;
  box-shadow: 0 4px 15px rgba(102, 126, 234, 0.4);
}

.submit-btn:hover:not(:disabled) {
  transform: translateY(-2px);
  box-shadow: 0 6px 20px rgba(102, 126, 234, 0.5);
}

.submit-btn:disabled {
  opacity: 0.7;
  cursor: not-allowed;
  transform: none;
}

.loading-text {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 10px;
}

.spinner-small {
  width: 16px;
  height: 16px;
  border: 2px solid rgba(255, 255, 255, 0.3);
  border-top: 2px solid white;
  border-radius: 50%;
  animation: spin 0.8s linear infinite;
}

/* Map Section */
.map-container {
  max-width: 1200px;
  margin: 0 auto 40px;
  padding: 0 20px;
}

.map {
  height: 450px;
  border-radius: 20px;
  box-shadow: 0 20px 60px rgba(0, 0, 0, 0.3);
  overflow: hidden;
}

/* Loading Overlay */
.loading-overlay {
  position: fixed;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  background: rgba(0, 0, 0, 0.7);
  backdrop-filter: blur(5px);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 1000;
}

.loading-card {
  background: white;
  padding: 40px;
  border-radius: 20px;
  text-align: center;
  max-width: 400px;
  box-shadow: 0 20px 60px rgba(0, 0, 0, 0.3);
}

.loading-spinner {
  width: 60px;
  height: 60px;
  border: 4px solid #f3f4f6;
  border-top: 4px solid #667eea;
  border-radius: 50%;
  animation: spin 1s linear infinite;
  margin: 0 auto 20px;
}

@keyframes spin {
  0% { transform: rotate(0deg); }
  100% { transform: rotate(360deg); }
}

.loading-card h3 {
  color: #1f2937;
  margin-bottom: 10px;
  font-size: 1.5rem;
}

.loading-card p {
  color: #6b7280;
}

/* Error Banner */
.error-banner {
  max-width: 800px;
  margin: 20px auto;
  padding: 20px 30px;
  background: #fee2e2;
  border: 2px solid #ef4444;
  border-radius: 12px;
  color: #991b1b;
  display: flex;
  align-items: center;
  gap: 15px;
  box-shadow: 0 4px 15px rgba(239, 68, 68, 0.2);
}

.error-icon {
  font-size: 1.5rem;
}

.close-btn {
  margin-left: auto;
  background: none;
  border: none;
  font-size: 1.5rem;
  cursor: pointer;
  color: #991b1b;
  padding: 0;
  width: 30px;
  height: 30px;
  display: flex;
  align-items: center;
  justify-content: center;
  border-radius: 50%;
  transition: background 0.3s;
}

.close-btn:hover {
  background: rgba(153, 27, 27, 0.1);
}

/* Results Container */
.results-container {
  max-width: 1200px;
  margin: 0 auto;
  padding: 0 20px;
}

.results-header {
  background: white;
  padding: 30px;
  border-radius: 20px;
  margin-bottom: 30px;
  box-shadow: 0 10px 40px rgba(0, 0, 0, 0.2);
  display: flex;
  justify-content: space-between;
  align-items: center;
  flex-wrap: wrap;
  gap: 20px;
}

.results-title {
  display: flex;
  align-items: center;
  gap: 15px;
  flex-wrap: wrap;
  font-size: 1.8rem;
}

.route-badge {
  background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
  color: white;
  padding: 10px 20px;
  border-radius: 12px;
  font-weight: 600;
}

.arrow {
  color: #667eea;
  font-size: 2rem;
}

.budget-display {
  display: flex;
  flex-direction: column;
  align-items: flex-end;
}

.budget-label {
  font-size: 0.9rem;
  color: #6b7280;
  text-transform: uppercase;
  letter-spacing: 1px;
}

.budget-amount {
  font-size: 2rem;
  font-weight: 700;
  color: #10b981;
}

/* Day Cards Grid */
.days-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(350px, 1fr));
  gap: 25px;
  margin-bottom: 30px;
}

.day-card {
  background: white;
  border-radius: 20px;
  overflow: hidden;
  box-shadow: 0 10px 40px rgba(0, 0, 0, 0.2);
  transition: transform 0.3s, box-shadow 0.3s;
  animation: slideIn 0.5s ease-out forwards;
  opacity: 0;
}

@keyframes slideIn {
  from {
    opacity: 0;
    transform: translateY(30px);
  }
  to {
    opacity: 1;
    transform: translateY(0);
  }
}

.day-card:hover {
  transform: translateY(-5px);
  box-shadow: 0 15px 50px rgba(0, 0, 0, 0.3);
}

.day-header {
  background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
  color: white;
  padding: 20px;
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.day-number {
  display: flex;
  flex-direction: column;
}

.day-label {
  font-size: 0.9rem;
  opacity: 0.9;
  text-transform: uppercase;
  letter-spacing: 1px;
}

.day-value {
  font-size: 2.5rem;
  font-weight: 800;
}

.day-total {
  display: flex;
  flex-direction: column;
  align-items: flex-end;
}

.total-label {
  font-size: 0.85rem;
  opacity: 0.9;
}

.total-amount {
  font-size: 1.3rem;
  font-weight: 700;
}

.day-content {
  padding: 25px;
}

.section-title {
  display: flex;
  align-items: center;
  gap: 8px;
  color: #1f2937;
  font-size: 1.1rem;
  margin-bottom: 15px;
  font-weight: 600;
}

.section-icon {
  font-size: 1.3rem;
}

.activities-section {
  margin-bottom: 25px;
}

.activities-text {
  color: #4b5563;
  line-height: 1.7;
  font-size: 1rem;
}

.expenses-section {
  background: #f9fafb;
  padding: 20px;
  border-radius: 12px;
}

.expenses-list {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.expense-item {
  display: grid;
  grid-template-columns: auto 1fr auto;
  align-items: center;
  gap: 10px;
  padding: 10px;
  background: white;
  border-radius: 8px;
  transition: transform 0.2s;
}

.expense-item:hover {
  transform: translateX(5px);
}

.expense-category {
  color: #374151;
  font-weight: 500;
  text-transform: capitalize;
}

.expense-dots {
  height: 1px;
  background: repeating-linear-gradient(
    to right,
    #d1d5db 0,
    #d1d5db 5px,
    transparent 5px,
    transparent 10px
  );
}

.expense-amount {
  color: #059669;
  font-weight: 700;
  font-size: 1.1rem;
}

/* Summary Card */
.summary-card {
  background: white;
  padding: 30px;
  border-radius: 20px;
  box-shadow: 0 10px 40px rgba(0, 0, 0, 0.2);
}

.summary-title {
  color: #1f2937;
  font-size: 1.8rem;
  margin-bottom: 25px;
  text-align: center;
}

.summary-stats {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
  gap: 20px;
}

.stat-item {
  background: linear-gradient(135deg, #f3f4f6 0%, #e5e7eb 100%);
  padding: 20px;
  border-radius: 12px;
  display: flex;
  flex-direction: column;
  align-items: center;
  text-align: center;
  gap: 8px;
}

.stat-icon {
  font-size: 2rem;
}

.stat-label {
  color: #6b7280;
  font-size: 0.9rem;
  text-transform: uppercase;
  letter-spacing: 1px;
}

.stat-value {
  color: #1f2937;
  font-size: 1.2rem;
  font-weight: 700;
}

/* Transitions */
.fade-enter-active, .fade-leave-active {
  transition: opacity 0.5s ease;
}

.fade-enter-from, .fade-leave-to {
  opacity: 0;
}

.slide-down-enter-active {
  animation: slideDown 0.4s ease-out;
}

.slide-down-leave-active {
  animation: slideDown 0.4s ease-out reverse;
}

@keyframes slideDown {
  from {
    opacity: 0;
    transform: translateY(-20px);
  }
  to {
    opacity: 1;
    transform: translateY(0);
  }
}

/* Responsive Design */
@media (max-width: 768px) {
  .title {
    font-size: 2rem;
  }
  
  .subtitle {
    font-size: 1rem;
  }
  
  .input-group {
    grid-template-columns: 1fr;
  }
  
  .map {
    height: 300px;
  }
  
  .days-grid {
    grid-template-columns: 1fr;
  }
  
  .results-header {
    flex-direction: column;
    text-align: center;
  }
  
  .budget-display {
    align-items: center;
  }
  
  .results-title {
    font-size: 1.3rem;
    justify-content: center;
  }
  
  .summary-stats {
    grid-template-columns: 1fr;
  }
}
 
</style>