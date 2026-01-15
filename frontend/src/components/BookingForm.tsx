import React, { useState } from 'react';

interface BookingFormData {
  guestId: string;
  roomId: string;
  checkIn: string;
  checkOut: string;
}

const BookingForm: React.FC = () => {
  const [formData, setFormData] = useState<BookingFormData>({
    guestId: '',
    roomId: '',
    checkIn: '',
    checkOut: '',
  });
  const [message, setMessage] = useState<string>('');
  const [isLoading, setIsLoading] = useState<boolean>(false);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setIsLoading(true);
    setMessage('');

    try {
      const response = await fetch('http://localhost:8080/api/bookings', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({
          guest_id: formData.guestId,
          room_id: formData.roomId,
          check_in: new Date(formData.checkIn).toISOString(),
          check_out: new Date(formData.checkOut).toISOString(),
        }),
      });

      if (response.ok) {
        const booking = await response.json();
        setMessage(`Booking created successfully! ID: ${booking.id}`);
        setFormData({
          guestId: '',
          roomId: '',
          checkIn: '',
          checkOut: '',
        });
      } else {
        const error = await response.text();
        setMessage(`Error: ${error}`);
      }
    } catch (error) {
      setMessage(`Network error: ${error}`);
    } finally {
      setIsLoading(false);
    }
  };

  const handleChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    const { name, value } = e.target;
    setFormData(prev => ({
      ...prev,
      [name]: value,
    }));
  };

  return (
    <div className="booking-form">
      <h2>Create a Booking</h2>
      <form onSubmit={handleSubmit}>
        <div className="form-group">
          <label htmlFor="guestId">Guest ID (UUID):</label>
          <input
            type="text"
            id="guestId"
            name="guestId"
            value={formData.guestId}
            onChange={handleChange}
            required
            placeholder="e.g., 123e4567-e89b-12d3-a456-426614174000"
          />
        </div>

        <div className="form-group">
          <label htmlFor="roomId">Room ID (UUID):</label>
          <input
            type="text"
            id="roomId"
            name="roomId"
            value={formData.roomId}
            onChange={handleChange}
            required
            placeholder="e.g., 987fcdeb-51a2-43f7-8d4c-a8b3e3456789"
          />
        </div>

        <div className="form-group">
          <label htmlFor="checkIn">Check-in Date:</label>
          <input
            type="date"
            id="checkIn"
            name="checkIn"
            value={formData.checkIn}
            onChange={handleChange}
            required
          />
        </div>

        <div className="form-group">
          <label htmlFor="checkOut">Check-out Date:</label>
          <input
            type="date"
            id="checkOut"
            name="checkOut"
            value={formData.checkOut}
            onChange={handleChange}
            required
          />
        </div>

        <button type="submit" disabled={isLoading}>
          {isLoading ? 'Creating...' : 'Create Booking'}
        </button>
      </form>

      {message && (
        <div className={`message ${message.includes('Error') ? 'error' : 'success'}`}>
          {message}
        </div>
      )}

      <style>{`
        .booking-form {
          background: rgba(var(--accent-dark), 0.5);
          padding: 2rem;
          border-radius: 8px;
          margin-top: 2rem;
        }

        h2 {
          margin-top: 0;
          margin-bottom: 1.5rem;
          color: white;
        }

        .form-group {
          margin-bottom: 1.5rem;
        }

        label {
          display: block;
          margin-bottom: 0.5rem;
          color: rgb(var(--accent-light));
          font-weight: 500;
        }

        input {
          width: 100%;
          padding: 0.75rem;
          border: 1px solid rgba(var(--accent-light), 0.3);
          border-radius: 4px;
          background: rgba(0, 0, 0, 0.3);
          color: white;
          font-size: 1rem;
          box-sizing: border-box;
        }

        input:focus {
          outline: none;
          border-color: rgb(var(--accent));
        }

        button {
          width: 100%;
          padding: 0.75rem;
          background: rgb(var(--accent));
          color: white;
          border: none;
          border-radius: 4px;
          font-size: 1rem;
          font-weight: 600;
          cursor: pointer;
          transition: background 0.2s;
        }

        button:hover:not(:disabled) {
          background: rgb(var(--accent-light));
          color: rgb(var(--accent-dark));
        }

        button:disabled {
          opacity: 0.6;
          cursor: not-allowed;
        }

        .message {
          margin-top: 1rem;
          padding: 1rem;
          border-radius: 4px;
          text-align: center;
        }

        .message.success {
          background: rgba(0, 255, 0, 0.1);
          border: 1px solid rgba(0, 255, 0, 0.3);
          color: #90EE90;
        }

        .message.error {
          background: rgba(255, 0, 0, 0.1);
          border: 1px solid rgba(255, 0, 0, 0.3);
          color: #FFB6C1;
        }
      `}</style>
    </div>
  );
};

export default BookingForm;
