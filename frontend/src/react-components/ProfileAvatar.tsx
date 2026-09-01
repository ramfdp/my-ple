import React, { useState } from 'react';
import { Camera, User } from 'lucide-react';

interface ProfileAvatarProps {
  src?: string;
  alt?: string;
  size?: number;
  borderWidth?: number;
  editable?: boolean;
  onEdit?: () => void;
  className?: string;
}

// ponytail: responsive avatar component with image error fallback to Lucide User icon
export const ProfileAvatar: React.FC<ProfileAvatarProps> = ({
  src,
  alt = 'Employee Avatar',
  size = 100,
  borderWidth = 4,
  editable = false,
  onEdit,
  className = '',
}) => {
  const [hasError, setHasError] = useState(false);

  return (
    <div
      className={`relative inline-flex items-center justify-center rounded-full bg-slate-100 border-white shadow-md flex-shrink-0 ${className}`}
      style={{
        width: `${size}px`,
        height: `${size}px`,
        borderWidth: `${borderWidth}px`,
      }}
    >
      {src && !hasError ? (
        <img
          src={src}
          alt={alt}
          onError={() => setHasError(true)}
          className="w-full h-full rounded-full object-cover"
        />
      ) : (
        <div className="w-full h-full rounded-full flex items-center justify-center bg-teal-50 text-teal-700">
          <User className="w-1/2 h-1/2" />
        </div>
      )}

      {editable && (
        <button
          type="button"
          onClick={onEdit}
          aria-label="Change profile photo"
          className="absolute bottom-0 right-0 w-8 h-8 rounded-full bg-teal-600 hover:bg-teal-700 text-white border-2 border-white flex items-center justify-center shadow-sm transition-all hover:scale-105"
        >
          <Camera className="w-4 h-4" />
        </button>
      )}
    </div>
  );
};
