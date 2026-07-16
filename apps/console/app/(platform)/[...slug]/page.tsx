import { notFound } from "next/navigation";

import { FeatureWorkspace } from "@/components/feature-workspace";
import { getNavigationItem } from "@/lib/navigation";

type FeaturePageProps = {
  params: Promise<{ slug: string[] }>;
};

export default async function FeaturePage({ params }: FeaturePageProps) {
  const { slug } = await params;
  const feature = getNavigationItem(`/${slug.join("/")}`);

  if (!feature || feature.href === "/dashboard" || feature.href === "/requests") {
    notFound();
  }

  return <FeatureWorkspace feature={feature} />;
}
