#!/usr/bin/env node
/**
 * DrogonClaw Skill Manifest Generator
 * Reads all TypeScript skill files and emits skills_manifest.json
 * Run: node scripts/gen_skill_manifest.mjs
 */

import { readFileSync, writeFileSync, readdirSync } from 'fs';
import { join, dirname } from 'path';
import { fileURLToPath } from 'url';

const __dirname = dirname(fileURLToPath(import.meta.url));
const skillsDir = join(__dirname, '..', 'skills', 'pentest');
const outputPath = join(__dirname, '..', 'skills_manifest.json');

const files = readdirSync(skillsDir)
  .filter(f => f.endsWith('.ts') && f !== 'index.ts');

const manifest = [];

for (const file of files) {
  const src = readFileSync(join(skillsDir, file), 'utf-8');
  
  // Extract tool name
  const nameMatch = src.match(/name:\s*["']([^"']+)["']/);
  if (!nameMatch) continue;
  const name = nameMatch[1];

  // Extract description (can be backtick or quote delimited)
  const descMatch = src.match(/description:\s*[`"']([^`"']{10,})[`"']/);
  const description = descMatch ? descMatch[1].trim().replace(/\s+/g, ' ') : `Execute ${name}`;

  // Extract parameters from zod schema - parse key names
  const parameters = {};
  
  // Look for z.object({ ... }) schema definition
  const schemaMatch = src.match(/schema:\s*z\.object\(\{([^}]+)\}\)/s);
  if (schemaMatch) {
    const schemaBody = schemaMatch[1];
    // Extract each field: fieldName: z.string().describe("...") or z.number() etc.
    const fieldRegex = /(\w+):\s*z\.(string|number|boolean|array|enum)[^,\n]*/g;
    let fieldMatch;
    while ((fieldMatch = fieldRegex.exec(schemaBody)) !== null) {
      const fieldName = fieldMatch[1];
      const fieldType = fieldMatch[2] === 'number' ? 'integer' : 
                        fieldMatch[2] === 'boolean' ? 'boolean' : 'string';
      
      // Try to find .describe() for this field
      const descRegex = new RegExp(`${fieldName}:[^\\n]+\\.describe\\(["'\`]([^"'\`]+)["'\`]\\)`);
      const fieldDescMatch = src.match(descRegex);
      const fieldDesc = fieldDescMatch ? fieldDescMatch[1] : `The ${fieldName} parameter`;
      
      // Check if optional
      const isOptional = src.includes(`${fieldName}: z.${fieldMatch[2]}`) && 
                         src.includes(`${fieldName}: z.${fieldMatch[2]}().optional()`);
      
      parameters[fieldName] = {
        type: fieldType,
        description: fieldDesc,
        required: !isOptional,
      };
    }
  }

  // If no params extracted, add a generic command param
  if (Object.keys(parameters).length === 0) {
    parameters['command'] = {
      type: 'string',
      description: 'The shell command or arguments to execute',
      required: true,
    };
  }

  manifest.push({
    name,
    description,
    parameters,
    executes_via: 'docker_shell',
  });
}

writeFileSync(outputPath, JSON.stringify(manifest, null, 2));
console.log(`[+] Generated skills_manifest.json with ${manifest.length} skills`);
manifest.forEach(s => console.log(`    - ${s.name}`));
