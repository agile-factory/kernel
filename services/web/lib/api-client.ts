import axios, { AxiosInstance, AxiosError } from 'axios'

class APIClient {
  private client: AxiosInstance

  constructor() {
    this.client = axios.create({
      baseURL: process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8080',
      timeout: 10000,
      headers: {
        'Content-Type': 'application/json',
      },
      withCredentials: true, // For HTTP-only cookies (refresh token)
    })

    // Request interceptor - Add access token
    this.client.interceptors.request.use(
      (config) => {
        const token = localStorage.getItem('access_token')
        if (token) {
          config.headers.Authorization = `Bearer ${token}`
        }
        return config
      },
      (error) => Promise.reject(error)
    )

    // Response interceptor - Handle token refresh
    this.client.interceptors.response.use(
      (response) => response,
      async (error: AxiosError) => {
        const originalRequest = error.config as any

        // If 401 and not already retrying, try to refresh token
        if (error.response?.status === 401 && !originalRequest._retry) {
          originalRequest._retry = true

          try {
            const { data } = await this.client.post('/api/v1/auth/refresh')
            localStorage.setItem('access_token', data.access_token)

            // Retry original request with new token
            originalRequest.headers.Authorization = `Bearer ${data.access_token}`
            return this.client(originalRequest)
          } catch (refreshError) {
            // Refresh failed, redirect to login
            localStorage.removeItem('access_token')
            if (typeof window !== 'undefined') {
              window.location.href = '/login'
            }
            return Promise.reject(refreshError)
          }
        }

        return Promise.reject(error)
      }
    )
  }

  // Auth endpoints
  async register(data: RegisterData) {
    const response = await this.client.post('/api/v1/auth/register', data)
    return response.data
  }

  async login(data: LoginData) {
    const response = await this.client.post('/api/v1/auth/login', data)
    localStorage.setItem('access_token', response.data.access_token)
    return response.data
  }

  async logout() {
    await this.client.post('/api/v1/auth/logout')
    localStorage.removeItem('access_token')
  }

  async forgotPassword(email: string) {
    const response = await this.client.post('/api/v1/auth/forgot-password', { email })
    return response.data
  }

  async resetPassword(token: string, password: string) {
    const response = await this.client.post('/api/v1/auth/reset-password', { token, password })
    return response.data
  }

  // Listings endpoints
  async getListings(params?: ListingSearchParams) {
    const response = await this.client.get('/api/v1/listings', { params })
    return response.data
  }

  async getListingById(id: string) {
    const response = await this.client.get(`/api/v1/listings/${id}`)
    return response.data
  }

  async createListing(data: CreateListingData) {
    const response = await this.client.post('/api/v1/listings', data)
    return response.data
  }

  async updateListing(id: string, data: UpdateListingData) {
    const response = await this.client.put(`/api/v1/listings/${id}`, data)
    return response.data
  }

  async deleteListing(id: string) {
    const response = await this.client.delete(`/api/v1/listings/${id}`)
    return response.data
  }

  // Bookings endpoints
  async getUserBookings() {
    const response = await this.client.get('/api/v1/bookings')
    return response.data
  }

  async getBookingById(id: string) {
    const response = await this.client.get(`/api/v1/bookings/${id}`)
    return response.data
  }

  async createBooking(data: CreateBookingData) {
    const response = await this.client.post('/api/v1/bookings', data)
    return response.data
  }

  async cancelBooking(id: string) {
    const response = await this.client.patch(`/api/v1/bookings/${id}/cancel`)
    return response.data
  }

  // Payments endpoints
  async createPayment(data: CreatePaymentData) {
    const response = await this.client.post('/api/v1/payments', data)
    return response.data
  }

  async getPaymentById(id: string) {
    const response = await this.client.get(`/api/v1/payments/${id}`)
    return response.data
  }

  // Favorites endpoints
  async getFavorites() {
    const response = await this.client.get('/api/v1/favorites')
    return response.data
  }

  async addFavorite(listingId: string) {
    const response = await this.client.post('/api/v1/favorites', { listing_id: listingId })
    return response.data
  }

  async removeFavorite(listingId: string) {
    const response = await this.client.delete(`/api/v1/favorites/${listingId}`)
    return response.data
  }
}

// Types
export interface RegisterData {
  email: string
  password: string
  first_name: string
  last_name: string
  phone?: string
  role: 'tenant' | 'landlord'
}

export interface LoginData {
  email: string
  password: string
}

export interface ListingSearchParams {
  city?: string
  min_price?: number
  max_price?: number
  min_surface?: number
  max_surface?: number
  bedrooms?: number
  bathrooms?: number
  property_type?: string
  amenities?: string[]
  lat?: number
  lon?: number
  radius_km?: number
  page?: number
  limit?: number
  sort_by?: string
}

export interface CreateListingData {
  title: string
  description: string
  property_type: string
  address: string
  city: string
  postal_code?: string
  latitude: number
  longitude: number
  price_per_month: number
  deposit_amount: number
  surface_area: number
  bedrooms: number
  bathrooms: number
  floor?: number
  total_floors?: number
  amenities: string[]
  available_from: string
  min_lease_months: number
  max_lease_months?: number
}

export interface UpdateListingData {
  title?: string
  description?: string
  status?: string
  price_per_month?: number
  deposit_amount?: number
  amenities?: string[]
  available_from?: string
  min_lease_months?: number
  max_lease_months?: number
}

export interface CreateBookingData {
  listing_id: string
  check_in: string
  check_out: string
}

export interface CreatePaymentData {
  booking_id: string
  amount: number
  payment_type: 'deposit' | 'balance'
  payment_method: 'wave' | 'orange_money' | 'free_money' | 'card'
}

export const apiClient = new APIClient()
