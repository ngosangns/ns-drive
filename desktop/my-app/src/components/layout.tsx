import React from "react";
import Link from "next/link";
import { usePathname } from "next/navigation";
import { Home, Settings, Cloud } from "lucide-react";
import { cn } from "@/lib/utils";
import { Button } from "./ui/button";

interface LayoutProps {
  children: React.ReactNode;
}

export function Layout({ children }: LayoutProps) {
  const pathname = usePathname();

  return (
    <div className="flex h-screen">
      {/* Sidebar */}
      <div className="bg-gray-900 w-48 p-0">
        <div className="flex flex-col items-center py-4 bg-primary">
          <h1 className="text-3xl font-bold select-none cursor-default text-white">
            Drive
          </h1>
        </div>
        <div className="p-3 flex flex-col gap-3">
          <Button
            variant={pathname === "/" ? "accent" : "ghost"}
            className="w-full justify-start"
            asChild
          >
            <Link href="/">
              <Home className="mr-2 h-4 w-4" />
              Home
            </Link>
          </Button>
          <Button
            variant={pathname === "/profiles" ? "accent" : "ghost"}
            className="w-full justify-start"
            asChild
          >
            <Link href="/profiles">
              <Settings className="mr-2 h-4 w-4" />
              Profiles
            </Link>
          </Button>
          <Button
            variant={pathname === "/remotes" ? "accent" : "ghost"}
            className="w-full justify-start"
            asChild
          >
            <Link href="/remotes">
              <Cloud className="mr-2 h-4 w-4" />
              Remotes
            </Link>
          </Button>
        </div>
      </div>

      {/* Main content */}
      <div className="bg-gray-900 flex-1 overflow-auto">{children}</div>
    </div>
  );
}
