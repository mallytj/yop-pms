<!-- src/routes/+page.svelte -->

<script lang="ts">
</script>

<main class="min-h-screen bg-gray-50">
  <!-- Top Navigation -->
  <nav class="sticky top-0 z-50 bg-white border-b border-gray-200 shadow-sm">
    <div class="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-4 flex justify-between items-center">
      <div class="flex items-center gap-3">
        <span class="text-3xl">🏨</span>
        <div>
          <span class="text-2xl font-bold text-gray-900">Ollerod</span>
          <p class="text-sm text-gray-500">Property Management</p>
        </div>
      </div>
      <div class="flex items-center gap-4">
        <button class="text-gray-600 hover:text-gray-900">🔔</button>
        <button class="text-gray-600 hover:text-gray-900">⚙️</button>
        <div class="w-10 h-10 rounded-full bg-blue-600 flex items-center justify-center text-white font-bold">M</div>
      </div>
    </div>
  </nav>

  <!-- Main Content -->
  <div class="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-8">
    <!-- Header -->
    <div class="mb-8">
      <h1 class="text-4xl font-bold text-gray-900 mb-2">Welcome Back</h1>
      <p class="text-gray-600">Here's what's happening with your property today</p>
    </div>

    <!-- Quick Stats -->
    <div class="grid md:grid-cols-4 gap-6 mb-8">
      <div class="bg-white rounded-xl p-6 border border-gray-200 hover:shadow-lg transition">
        <div class="flex justify-between items-start mb-4">
          <span class="text-3xl">🛏️</span>
          <span class="text-sm font-semibold text-green-600 bg-green-100 px-3 py-1 rounded-full">12/15</span>
        </div>
        <p class="text-gray-600 text-sm">Rooms Occupied</p>
        <p class="text-2xl font-bold text-gray-900">80%</p>
      </div>

      <div class="bg-white rounded-xl p-6 border border-gray-200 hover:shadow-lg transition">
        <div class="flex justify-between items-start mb-4">
          <span class="text-3xl">📅</span>
          <span class="text-sm font-semibold text-blue-600 bg-blue-100 px-3 py-1 rounded-full">+3</span>
        </div>
        <p class="text-gray-600 text-sm">Today's Arrivals</p>
        <p class="text-2xl font-bold text-gray-900">3 Guests</p>
      </div>

      <div class="bg-white rounded-xl p-6 border border-gray-200 hover:shadow-lg transition">
        <div class="flex justify-between items-start mb-4">
          <span class="text-3xl">💰</span>
          <span class="text-sm font-semibold text-amber-600 bg-amber-100 px-3 py-1 rounded-full">Today</span>
        </div>
        <p class="text-gray-600 text-sm">Revenue</p>
        <p class="text-2xl font-bold text-gray-900">$2,400</p>
      </div>

      <div class="bg-white rounded-xl p-6 border border-gray-200 hover:shadow-lg transition">
        <div class="flex justify-between items-start mb-4">
          <span class="text-3xl">⚠️</span>
          <span class="text-sm font-semibold text-red-600 bg-red-100 px-3 py-1 rounded-full">1</span>
        </div>
        <p class="text-gray-600 text-sm">Pending Tasks</p>
        <p class="text-2xl font-bold text-gray-900">1 Issue</p>
      </div>
    </div>

    <!-- Main Grid -->
    <div class="grid lg:grid-cols-3 gap-8">
      <!-- Left Column -->
      <div class="lg:col-span-2">
        <!-- Quick Actions -->
        <div class="bg-white rounded-xl border border-gray-200 p-6 mb-8">
          <h2 class="text-xl font-bold text-gray-900 mb-4">Quick Actions</h2>
          <div class="grid grid-cols-2 sm:grid-cols-4 gap-4">
            <a href="/planner" class="flex flex-col items-center justify-center p-4 rounded-lg border-2 border-gray-200 hover:border-blue-500 hover:bg-blue-50 transition group">
              <span class="text-3xl mb-2 group-hover:scale-110 transition">📅</span>
              <span class="text-sm font-semibold text-gray-900">Planner</span>
            </a>
            
            <button class="flex flex-col items-center justify-center p-4 rounded-lg border-2 border-gray-200 hover:border-green-500 hover:bg-green-50 transition group">
              <span class="text-3xl mb-2 group-hover:scale-110 transition">➕</span>
              <span class="text-sm font-semibold text-gray-900">New Guest</span>
            </button>
            
            <button class="flex flex-col items-center justify-center p-4 rounded-lg border-2 border-gray-200 hover:border-purple-500 hover:bg-purple-50 transition group">
              <span class="text-3xl mb-2 group-hover:scale-110 transition">🔑</span>
              <span class="text-sm font-semibold text-gray-900">Check-In</span>
            </button>
            
            <button class="flex flex-col items-center justify-center p-4 rounded-lg border-2 border-gray-200 hover:border-orange-500 hover:bg-orange-50 transition group">
              <span class="text-3xl mb-2 group-hover:scale-110 transition">📊</span>
              <span class="text-sm font-semibold text-gray-900">Reports</span>
            </button>
          </div>
        </div>

        <!-- Recent Bookings -->
        <div class="bg-white rounded-xl border border-gray-200 p-6">
          <div class="flex justify-between items-center mb-4">
            <h2 class="text-xl font-bold text-gray-900">Recent Bookings</h2>
            <a href="/planner" class="text-blue-600 hover:text-blue-700 text-sm font-semibold">View All →</a>
          </div>
          
          <div class="space-y-4">
            <div class="flex items-center justify-between p-4 bg-gray-50 rounded-lg hover:bg-gray-100 transition cursor-pointer">
              <div class="flex items-center gap-4">
                <div class="w-12 h-12 rounded-full bg-blue-200 flex items-center justify-center font-bold text-blue-700">JD</div>
                <div>
                  <p class="font-semibold text-gray-900">John Doe</p>
                  <p class="text-sm text-gray-600">Room 101 • 2 nights</p>
                </div>
              </div>
              <div class="text-right">
                <p class="font-semibold text-gray-900">$480</p>
                <p class="text-sm text-green-600">Confirmed</p>
              </div>
            </div>

            <div class="flex items-center justify-between p-4 bg-gray-50 rounded-lg hover:bg-gray-100 transition cursor-pointer">
              <div class="flex items-center gap-4">
                <div class="w-12 h-12 rounded-full bg-purple-200 flex items-center justify-center font-bold text-purple-700">SM</div>
                <div>
                  <p class="font-semibold text-gray-900">Sarah Miller</p>
                  <p class="text-sm text-gray-600">Room 205 • 3 nights</p>
                </div>
              </div>
              <div class="text-right">
                <p class="font-semibold text-gray-900">$720</p>
                <p class="text-sm text-green-600">Confirmed</p>
              </div>
            </div>

            <div class="flex items-center justify-between p-4 bg-gray-50 rounded-lg hover:bg-gray-100 transition cursor-pointer">
              <div class="flex items-center gap-4">
                <div class="w-12 h-12 rounded-full bg-green-200 flex items-center justify-center font-bold text-green-700">TC</div>
                <div>
                  <p class="font-semibold text-gray-900">Tom Chen</p>
                  <p class="text-sm text-gray-600">Room 302 • 1 night</p>
                </div>
              </div>
              <div class="text-right">
                <p class="font-semibold text-gray-900">$200</p>
                <p class="text-sm text-yellow-600">Pending</p>
              </div>
            </div>
          </div>
        </div>
      </div>

      <!-- Right Column -->
      <div>
        <!-- Room Status -->
        <div class="bg-white rounded-xl border border-gray-200 p-6 mb-8">
          <h2 class="text-xl font-bold text-gray-900 mb-4">Room Status</h2>
          
          <div class="space-y-3">
            <div class="flex items-center justify-between">
              <span class="text-sm font-semibold text-gray-900">Available</span>
              <span class="text-lg font-bold text-green-600">3</span>
            </div>
            <div class="w-full bg-gray-200 rounded-full h-2">
              <div class="bg-green-500 h-2 rounded-full" style="width: 20%"></div>
            </div>

            <div class="flex items-center justify-between pt-2">
              <span class="text-sm font-semibold text-gray-900">Occupied</span>
              <span class="text-lg font-bold text-blue-600">12</span>
            </div>
            <div class="w-full bg-gray-200 rounded-full h-2">
              <div class="bg-blue-500 h-2 rounded-full" style="width: 80%"></div>
            </div>

            <div class="flex items-center justify-between pt-2">
              <span class="text-sm font-semibold text-gray-900">Maintenance</span>
              <span class="text-lg font-bold text-orange-600">0</span>
            </div>
            <div class="w-full bg-gray-200 rounded-full h-2">
              <div class="bg-orange-500 h-2 rounded-full" style="width: 0%"></div>
            </div>
          </div>
        </div>

        <!-- Upcoming Check-outs -->
        <div class="bg-white rounded-xl border border-gray-200 p-6">
          <h2 class="text-xl font-bold text-gray-900 mb-4">Today's Check-outs</h2>
          
          <div class="space-y-3">
            <div class="p-3 bg-red-50 border border-red-200 rounded-lg">
              <p class="text-sm font-semibold text-gray-900">Room 105</p>
              <p class="text-xs text-gray-600">Check-out: 11:00 AM</p>
            </div>

            <div class="p-3 bg-red-50 border border-red-200 rounded-lg">
              <p class="text-sm font-semibold text-gray-900">Room 203</p>
              <p class="text-xs text-gray-600">Check-out: 11:00 AM</p>
            </div>

            <div class="p-3 bg-blue-50 border border-blue-200 rounded-lg">
              <p class="text-sm font-semibold text-gray-900">Room 301</p>
              <p class="text-xs text-gray-600">Check-in: 3:00 PM</p>
            </div>
          </div>
        </div>
      </div>
    </div>
  </div>
</main>

<style>
  @import "tailwindcss";
  main {
    background: linear-gradient(135deg, #f9fafb 0%, #f3f4f6 100%);
  }
</style>