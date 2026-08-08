import { useQuery } from "@tanstack/react-query";
import { authApi } from "../api/auth";

export const currentUserQueryKey = ["current-user"] as const;

export function useCurrentUser() {
  return useQuery({
    queryKey: currentUserQueryKey,
    queryFn: authApi.me,
    staleTime: 5 * 60 * 1000,
  });
}
