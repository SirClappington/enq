import Link from 'next/link'
async function fetchJobs(){
const res = await fetch(process.env.NEXT_PUBLIC_API_BASE+"/v1/jobs?limit=50", { cache: 'no-store' })
return res.json()
}
export default async function Home(){
const { jobs = [] } = await fetchJobs()
return (
<main className="p-6">
<h1 className="text-2xl font-semibold">Workerbee Dashboard</h1>
<table className="mt-6 w-full text-sm">
<thead><tr><th>ID</th><th>Type</th><th>Status</th><th>Attempt</th><th>Run At</th></tr></thead>
<tbody>
{jobs.map((j:any)=> (
<tr key={j.id} className="border-t">
<td><Link href={`/jobs/${j.id}`}>{j.id.slice(0,8)}</Link></td>
<td>{j.type}</td>
<td>{j.status}</td>
<td>{j.attempt}</td>
<td>{new Date(j.run_at).toLocaleString()}</td>
</tr>
))}
</tbody>
</table>
</main>
)
}